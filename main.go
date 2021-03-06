package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/gin-gonic/gin"
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
