package main

import (
	"context"
	"encoding/json"

	"github.com/qri-io/jsonschema"
)

// ValidateAgainstSchema validates JSON input against the corresponding JSON schema
func ValidateAgainstSchema(schema []byte, input []byte) []jsonschema.KeyError {
	ctx := context.Background()

	rs := &jsonschema.Schema{}
	if err := json.Unmarshal(schema, rs); err != nil {
		panic("unmarshal schema: " + err.Error())
	}

	errs, err := rs.ValidateBytes(ctx, input)
	if err != nil {
		panic(err)
	}

	return errs
}
