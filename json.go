package main

import (
	"encoding/json"
)

// ToJSONArray Converts an array of JSON encoded objects to an JSON array of objects
func ToJSONArray(data [][]byte) ([]byte, error) {
	var err error
	result := make([]map[string]interface{}, len(data))

	for index, item := range data {
		var res map[string]interface{}
		err = json.Unmarshal(item, &res)
		result[index] = res
	}

	jsonObject, err := json.Marshal(result)

	return jsonObject, err
}
