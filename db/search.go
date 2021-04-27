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

	var results [][]SeekResult

	maxResults := body.MaxResults

	if maxResults == nil {
		var d int = 100
		maxResults = &d
	}

	for _, term := range terms {

		split := strings.Split(strings.ReplaceAll(term, " ", ""), ":")

		fieldName := split[0]
		fieldValue := split[1]

		// Remove the / at the end of the key to search on values starting with the specified fieldValue
		// keyPrefix := []byte(fmt.Sprintf("streams@%v/fields/%v/%v", body.DataStream, fieldName, fieldValue))

		keyPrefix := []byte(fmt.Sprintf("streams@%v/fields/%v/%v/", body.DataStream, fieldName, fieldValue))

		items, err := seekItems(db, keyPrefix, *maxResults)

		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		results = append(results, items)
	}

	items := merge(results)[:*maxResults] // merge lists and slice the slice to desired size

	txn := db.NewTransaction(false)

	documentValues := make([][]byte, 0)

	for _, item := range items {
		document, err := findDocument(txn, item)

		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		documentValues = append(documentValues, document.Value)
	}

	data, err := ToJSONArray(documentValues)

	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Header("Content-Type", "application/json")
	c.Writer.Write(data)
}

type Document struct {
	SeekResultKey string
	Key           string
	Timestamp     int
	Value         []byte
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

func merge(items [][]SeekResult) []SeekResult {
	var concatList []SeekResult

	for _, sub := range items {
		for _, item := range sub {
			concatList = append(concatList, item)
		}
	}

	sort.Slice(concatList, func(i, j int) bool {
		return concatList[i].Timestamp > concatList[j].Timestamp
	})

	return concatList
}

type SeekResult struct {
	Key       string
	Timestamp int
	Value     []byte
}

func findDocument(db *badger.Txn, seekResult SeekResult) (*Document, error) {
	result, err := db.Get(seekResult.Value)

	if err != nil {
		return nil, err
	}

	var document *Document

	err = result.Value(func(v []byte) error {
		doc := Document{Key: string(result.Key()), SeekResultKey: seekResult.Key, Value: v, Timestamp: getTimestamp(seekResult.Key)}
		document = &doc
		return nil
	})

	if err != nil {
		return nil, err
	}

	return document, nil
}

func seekItems(db *badger.DB, keyPrefix []byte, maxResults int) ([]SeekResult, error) {
	var items []SeekResult

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

			item.Value(func(v []byte) error {
				itemKey := string(item.Key())
				items = append(items, SeekResult{Key: itemKey, Value: v, Timestamp: getTimestamp(itemKey)})
				return nil
			})

		}
		return nil
	})

	if err != nil {
		return items, err
	}

	return items, nil
}
