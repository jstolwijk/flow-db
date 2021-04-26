package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/gin-gonic/gin"
	"github.com/qri-io/jsonschema"
)

type searchCommand struct {
	DataStream string `json:"dataStream"`
	Query      string `json:"query"`
	MaxResults *int   `json:"maxResults"`
}
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

	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "UP",
		})
	})

	ctx := context.Background()

	r.POST("/api/configurations", func(c *gin.Context) {
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

	r.GET("/api/configurations/current", func(c *gin.Context) {
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
	r.POST("/api/data-streams/:streamName/documents", func(c *gin.Context) {
		streamName := c.Param("streamName")

		var body []map[string]interface{}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		wb := db.NewWriteBatch()
		defer wb.Cancel()

		for _, i := range body {

			id, err := seq.Next()

			if err != nil {
				c.Status(http.StatusInternalServerError)
				return
			}

			i["_id"] = id
			i["_dataStream"] = streamName
			i["_timestamp"] = time.Now().Unix()

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
			documentKey := fmt.Sprintf("streams@%v/documents/%v", streamName, id)

			err = wb.Set([]byte(documentKey), rawBody)

			if err != nil {
				c.Status(http.StatusInternalServerError)
				return
			}

			// Time based search
			key := fmt.Sprintf("streams@%v/indices/%d/%v", streamName, i["_timestamp"], id)
			err = wb.Set([]byte(key), []byte(documentKey))

			if err != nil {
				c.Status(http.StatusInternalServerError)
				return
			}

			// Index all fields
			for fieldName, fieldValue := range i {
				// "streams@<streamName>/fields/<fieldName>/<fieldValue>/<timestamp>/<documentId>"
				dbKey := fmt.Sprintf("streams@%v/fields/%v/%v/%d/%v", streamName, fieldName, fieldValue, i["_timestamp"], id)
				err = wb.Set([]byte(dbKey), []byte(documentKey))
			}

			if err != nil {
				c.Status(http.StatusInternalServerError)
				return
			}
		}

		wb.Flush()

		c.JSON(http.StatusCreated, gin.H{
			"_id": "todo",
		})
	})

	r.POST("/api/search", func(c *gin.Context) {
		var body searchCommand
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		terms := strings.Split(body.Query, " OR ")

		var results [][]ItemDTO

		for _, term := range terms {

			split := strings.Split(strings.ReplaceAll(term, " ", ""), ":")

			fieldName := split[0]
			fieldValue := split[1]

			// Remove the / at the end of the key to search on values starting with the specified fieldValue
			// keyPrefix := []byte(fmt.Sprintf("streams@%v/fields/%v/%v", body.DataStream, fieldName, fieldValue))

			keyPrefix := []byte(fmt.Sprintf("streams@%v/fields/%v/%v/", body.DataStream, fieldName, fieldValue))

			maxResults := body.MaxResults

			if maxResults == nil {
				var d int = 100
				maxResults = &d
			}

			items, err := seekItems(db, keyPrefix, *maxResults)

			if err != nil {
				c.Status(http.StatusInternalServerError)
				return
			}

			results = append(results, items)
		}

		items := merge(results)

		data, err := ToJSONArray(items)

		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		c.Header("Content-Type", "application/json")
		c.Writer.Write(data)
	})

	r.GET("/api/data-streams/:streamName/schema", func(c *gin.Context) {
		streamName := c.Param("streamName")

		var item *badger.Item
		err = db.View(func(txn *badger.Txn) error {
			key := fmt.Sprintf("streams@%v/schema", streamName)
			item, err = txn.Get([]byte(key))
			return err
		})

		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		err := item.Value(func(data []byte) error {
			c.Header("Content-Type", "application/json")
			c.Writer.Write(data)

			return nil
		})

		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
	})

	r.GET("/api/data-streams/:streamName/recent", func(c *gin.Context) {
		streamName := c.Param("streamName")
		query := c.Request.URL.Query()
		descending := query.Get("order") == "DESC"

		keyPrefix := []byte(fmt.Sprintf("streams@%v/indices/", streamName))

		startKey := keyPrefix

		if descending {
			startKey = append([]byte(fmt.Sprintf("streams@%v/indices/", streamName)), 0xFF)
		}

		items := make([][]byte, 0)

		db.View(func(txn *badger.Txn) error {

			// Explanation how ordering works: https://github.com/dgraph-io/badger/issues/347
			it := txn.NewIterator(badger.IteratorOptions{
				PrefetchValues: true,
				PrefetchSize:   100,
				Reverse:        descending,
			})
			defer it.Close()

			numberOfItems := 0

			// TODO: Make sure this is thread safe
			for it.Seek(startKey); it.ValidForPrefix(keyPrefix); it.Next() {
				item := it.Item()

				if numberOfItems >= 100 {
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

	r.GET("/api/data-streams/:streamName/documents/:documentID", func(c *gin.Context) {
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

type ItemDTO struct {
	Key       string
	Timestamp int
	Value     []byte
}

func getTimestamp(key string) int {
	split := strings.Split(key, "/")
	time := split[len(split)-2]
	i, err := strconv.Atoi(time)

	if err != nil {
		panic(err)
	}

	return i
}

func merge(items [][]ItemDTO) [][]byte {
	var concatList []ItemDTO

	for _, sub := range items {
		for _, item := range sub {
			concatList = append(concatList, item)
		}
	}

	sort.Slice(concatList, func(i, j int) bool {
		fmt.Println(concatList[i].Key)

		return concatList[i].Timestamp > concatList[j].Timestamp
	})

	var finalList [][]byte
	for _, item := range concatList {
		finalList = append(finalList, item.Value)
	}

	return finalList
}

func seekItems(db *badger.DB, keyPrefix []byte, maxResults int) ([]ItemDTO, error) {
	var items []ItemDTO

	err := db.View(func(txn *badger.Txn) error {

		// Explanation how ordering works: https://github.com/dgraph-io/badger/issues/347
		it := txn.NewIterator(badger.IteratorOptions{
			PrefetchValues: true,
			PrefetchSize:   maxResults,
			Reverse:        false,
		})
		defer it.Close()

		numberOfItems := 0

		// TODO: Make sure this is thread safe
		for it.Seek(keyPrefix); it.ValidForPrefix(keyPrefix); it.Next() {
			item := it.Item()

			if numberOfItems >= maxResults {
				return nil
			}

			numberOfItems++

			err := item.Value(func(v []byte) error {
				// TODO: Move retrieving of the document to the end of the /api/search function
				document, err := txn.Get(v)

				if err != nil {
					return err
				}

				return document.Value(func(v []byte) error {
					itemKey := string(item.Key())
					items = append(items, ItemDTO{Key: itemKey, Value: v, Timestamp: getTimestamp(itemKey)})

					return nil
				})
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return items, err
	}

	return items, nil
}
