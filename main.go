package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/gin-gonic/gin"
	"github.com/qri-io/jsonschema"
)

type FlowDatabase struct {
	inMemory bool
}

func main() {
	// TODO: create 2 databases
	// 1 in memory and 1 with disk access
	// Use in memory db as index cache
	// load index cache from disk on startup

	// Run database in memory for testing
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	seq, err := db.GetSequence([]byte("id"), 1000)
	defer seq.Release()

	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "UP",
		})
	})

	ctx := context.Background()

	r.POST("/index", func(c *gin.Context) {
		var body interface{}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		rawBody, err := json.Marshal(body)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate
		var schemaData = []byte(`{
			"$schema": "https://json-schema.org/draft/2019-09/schema",
			"type": "object",
			"properties": {
			  "type": {
				"type": "string"
			  },
			  "quantity": {
				"type": "integer"
			  },
			  "quality": {
				"type": "string",
				"enum": ["AAA", "A", "B", "C"]
			  },
			  "owner": {
				"type": "string"
			  }
			},
			"required": ["type", "quantity", "quality"]
		  }
		  `)

		rs := &jsonschema.Schema{}
		if err := json.Unmarshal(schemaData, rs); err != nil {
			panic("unmarshal schema: " + err.Error())
		}

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
			key := fmt.Sprintf("item-%d", id)

			return txn.Set([]byte(key), rawBody)
		})

		c.JSON(http.StatusCreated, gin.H{
			"_id": id,
		})
	})
	r.GET("/documents", func(c *gin.Context) {
		q := c.Request.URL.Query()
		id := q["id"][0]

		var item *badger.Item
		err := db.View(func(txn *badger.Txn) error {
			key := "item-" + id

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
