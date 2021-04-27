package main

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/gin-gonic/gin"
)

func search(db *badger.DB, c *gin.Context) {
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
