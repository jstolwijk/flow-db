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

type SortingDirection string

const (
	Ascending  SortingDirection = "ascending"
	Descending SortingDirection = "descending"
)

type searchCommand struct {
	DataStream       string           `json:"dataStream"`
	Query            string           `json:"query"`
	MaxResults       *int             `json:"maxResults"`
	SortingDirection SortingDirection `json:"sortingDirection"`
}

func search(db *badger.DB, c *gin.Context) {
	var body searchCommand
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	terms := parse(body.Query)

	var results [][]SeekResult

	maxResults := body.MaxResults

	if maxResults == nil {
		var d int = 100
		maxResults = &d
	}

	for _, or := range terms.Or {
		fmt.Println("Orrrr")
		for _, and := range or.And {
			fmt.Println("Anddd")
			if and.Operand != nil {
				fmt.Println("Yeah budy")

				fieldName := and.Operand.Operand.Summand[0].LHS.LHS.SymbolRef.Symbol
				fieldValue := and.Operand.ConditionRHS.Compare.Operand.Summand[0].LHS.LHS.Value.Number

				fmt.Println(fmt.Sprintf("fieldName: %v, fieldValue: %v", fieldName, *fieldValue))

				// Remove the / at the end of the key to search on values starting with the specified fieldValue
				// keyPrefix := []byte(fmt.Sprintf("streams@%v/fields/%v/%v", body.DataStream, fieldName, fieldValue))
				keyPrefix := []byte(fmt.Sprintf("streams@%v/fields/%v/%v/", body.DataStream, fieldName, *fieldValue))

				items, err := seekItems(db, keyPrefix, *maxResults, body.SortingDirection)

				if err != nil {
					fmt.Println("Error ", err)
					c.Status(http.StatusInternalServerError)
					return
				}

				results = append(results, items)
			} else {
				panic("Not is not implemented")
			}
		}
	}

	mergedResults := merge(results, body.SortingDirection) // merge lists and slice the slice to desired size

	nrOfResultsToReturn := min(*maxResults, len(mergedResults))
	items := mergedResults[:nrOfResultsToReturn]

	txn := db.NewTransaction(false)

	documentValues := make([][]byte, 0)

	for _, item := range items {
		document, err := findDocument(txn, item)

		if err != nil {
			fmt.Println("Error ", err)
			c.Status(http.StatusInternalServerError)
			return
		}

		documentValues = append(documentValues, document.Value)
	}

	data, err := ToJSONArray(documentValues)

	if err != nil {
		fmt.Println("Error ", err)
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

func merge(items [][]SeekResult, sortingDirection SortingDirection) []SeekResult {
	var concatList []SeekResult

	for _, sub := range items {
		for _, item := range sub {
			concatList = append(concatList, item)
		}
	}

	if sortingDirection == Descending {
		sort.Slice(concatList, func(i, j int) bool {
			return concatList[i].Timestamp > concatList[j].Timestamp
		})
	} else {
		sort.Slice(concatList, func(i, j int) bool {
			return concatList[i].Timestamp < concatList[j].Timestamp
		})
	}

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
		fmt.Println(fmt.Sprintf("Could not find document: %s ref: %s", string(seekResult.Value), seekResult.Key))
		return nil, err
	}

	var document *Document

	err = result.Value(func(v []byte) error {
		doc := Document{Key: string(result.Key()), SeekResultKey: seekResult.Key, Value: v, Timestamp: getTimestamp(seekResult.Key)}
		document = &doc
		return nil
	})

	if err != nil {
		fmt.Println("Error ", seekResult)
		return nil, err
	}

	return document, nil
}

func seekItems(db *badger.DB, keyPrefix []byte, maxResults int, sortingDirection SortingDirection) ([]SeekResult, error) {
	var items []SeekResult

	err := db.View(func(txn *badger.Txn) error {

		startKey := keyPrefix

		if sortingDirection == Descending {
			startKey = append(keyPrefix, 0xFF)
		}

		it := txn.NewIterator(badger.IteratorOptions{
			PrefetchValues: true,
			PrefetchSize:   maxResults,
			Reverse:        sortingDirection == Descending,
		})

		defer it.Close()

		numberOfItems := 0

		// TODO: Make sure this is thread safe
		for it.Seek(startKey); it.ValidForPrefix(keyPrefix); it.Next() {
			item := it.Item()
			key := string(item.Key())

			if numberOfItems >= maxResults {
				return nil
			}

			numberOfItems++

			err := item.Value(func(value []byte) error {
				/*
				 * TODO FIX THIS SHIT
				 * Results where fucked since this fucking badger db client will just override the same address space over and over again
				 * To fix this shit i've decided to make a deep copy of the value in a different part of memory
				 * code is supper inefficient right now, consider rewrite using channels :)
				 */
				var b = make([]byte, len(value))
				copy(b, value)

				items = append(items, SeekResult{Key: key, Value: b, Timestamp: getTimestamp(key)})
				return nil
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
