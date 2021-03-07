package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/gin-gonic/gin"
	"github.com/qri-io/jsonschema"
)

type newConfigurationCommand struct {
	DataStreams []dataStream `json:"dataStreams"`
}

type dataStream struct {
	Name   string      `json:"name"`
	Schema interface{} `json:"schema"`
}

func main() {
	// TODO: create 2 databases
	// 1 in memory and 1 with disk access
	// Use in memory db as index cache
	// load index cache from disk on startup

	// Run database in memory for testing
	db, err := badger.Open(badger.DefaultOptions("C:\\Users\\Jesse\\Projects\\flow-db\\data").WithInMemory(false))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// TODO: create sequence per index and create a deffered cleanup function
	seq, err := db.GetSequence([]byte("id"), 1000)
	defer seq.Release()

	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "UP",
		})
	})

	ctx := context.Background()

	r.POST("/configurations", func(c *gin.Context) {
		var body newConfigurationCommand
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err = db.Update(func(txn *badger.Txn) error {
			dataStreamNames := make([]string, len(body.DataStreams))

			var schema []byte
			for index, dataStream := range body.DataStreams {
				fmt.Printf("Creating index %v\n", dataStream.Name)
				dataStreamNames[index] = dataStream.Name

				key := fmt.Sprintf("streams@%v/schema", dataStream.Name)
				schema, err = json.Marshal(dataStream.Schema)
				err = txn.Set([]byte(key), schema)
			}

			rawBody, err := json.Marshal(dataStreamNames)

			if err != nil {
				return err
			}

			return txn.Set([]byte("streams"), rawBody)
		})

		c.Status(http.StatusAccepted)
	})

	r.GET("/configurations/current", func(c *gin.Context) {
		var item *badger.Item
		err := db.View(func(txn *badger.Txn) error {
			item, err = txn.Get([]byte("streams"))
			return err
		})

		if err != nil {
			c.Status(http.StatusInternalServerError)
		}

		if item != nil {
			item.Value(func(val []byte) error {
				c.Header("Content-Type", "application/json")
				c.Writer.Write(val)
				return nil
			})
		} else {
			c.Status(http.StatusInternalServerError)
		}
	})

	// Changed api to post multiple requests at a time atm
	r.POST("/data-streams/:streamName/documents", func(c *gin.Context) {
		streamName := c.Param("streamName")

		var body []interface{}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		for _, i := range body {

			rawBody, err := json.Marshal(i)

			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			// Validate
			var item *badger.Item
			err = db.View(func(txn *badger.Txn) error {
				key := fmt.Sprintf("streams@%v/schema", streamName)
				item, err = txn.Get([]byte(key))
				return err
			})

			if err != nil {
				c.Status(http.StatusInternalServerError)
			}

			rs := &jsonschema.Schema{}

			err = item.Value(func(val []byte) error {
				if err := json.Unmarshal(val, rs); err != nil {
					panic("unmarshal schema: " + err.Error())
				}

				return err
			})

			errs, err := rs.ValidateBytes(ctx, rawBody)
			if err != nil {
				panic(err)
			}

			if len(errs) > 0 {
				fmt.Println(errs[0].Error())
				c.JSON(http.StatusBadRequest, errs)
				return
			}

			// Store
			id, err := seq.Next()

			if err != nil {
				c.Status(http.StatusInternalServerError)
				return
			}

			err = db.Update(func(txn *badger.Txn) error {
				documentKey := fmt.Sprintf("streams@%v/documents/%v", streamName, id)

				err := txn.Set([]byte(documentKey), rawBody)

				if err != nil {
					return err
				}

				secs := time.Now().Unix()
				key := fmt.Sprintf("streams@%v/indices/%d/%v", streamName, secs, id)
				err = txn.Set([]byte(key), []byte(documentKey))

				return err
			})

		}
		c.JSON(http.StatusCreated, gin.H{
			"_id": "todo",
		})
	})

	r.GET("/data-streams/:streamName/recent", func(c *gin.Context) {
		streamName := c.Param("streamName")
		query := c.Request.URL.Query()
		ascending := query.Get("order") == "ASC"

		keyPrefix := []byte(fmt.Sprintf("streams@%v/indices/", streamName))

		startKey := keyPrefix

		if ascending {
			startKey = append([]byte(fmt.Sprintf("streams@%v/indices/", streamName)), 0xFF)
		}

		items := make([][]byte, 0)

		db.View(func(txn *badger.Txn) error {

			// Explanation how ordering works: https://github.com/dgraph-io/badger/issues/347
			it := txn.NewIterator(badger.IteratorOptions{
				PrefetchValues: true,
				PrefetchSize:   100,
				Reverse:        ascending,
			})
			defer it.Close()

			numberOfItems := 0

			// TODO: Make sure this is thread safe
			for it.Seek(startKey); it.ValidForPrefix(keyPrefix); it.Next() {
				item := it.Item()

				if numberOfItems >= 500 {
					return nil
				}

				numberOfItems++

				err := item.Value(func(v []byte) error {
					item, err = txn.Get(v)

					if err != nil {
						return err
					}

					return item.Value(func(v []byte) error {
						items = append(items, v)

						return nil
					})
				})
				if err != nil {
					return err
				}
			}
			return nil
		})

		data, err := ToJSONArray(items)

		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		c.Header("Content-Type", "application/json")
		c.Writer.Write(data)
	})

	r.GET("/data-streams/:streamName/documents/:documentID", func(c *gin.Context) {
		streamName := c.Param("streamName")
		documentID := c.Param("documentID")

		var item *badger.Item
		err := db.View(func(txn *badger.Txn) error {
			key := fmt.Sprintf("streams@%v/documents/%v", streamName, documentID)

			item, err = txn.Get([]byte(key))

			return err
		})

		if err != nil {
			c.Status(http.StatusInternalServerError)
		}

		if item != nil {
			item.Value(func(val []byte) error {
				c.Header("Content-Type", "application/json")
				c.Writer.Write(val)
				return nil
			})
		} else {
			c.Status(http.StatusInternalServerError)
		}
	})
	r.Run()

	// Your code here…
}
