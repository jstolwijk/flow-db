- configure indices/data stream via "config"/"migration" file
- index templates
- create diff tool to diff index config files
- create endpoint to apply new config
- implement search api
- allow to search on "keywords"
- create "stream" api
- expose grpc api from db app
- move rest api into separate package
- use batch write to improve indexing performance https://github.com/dgraph-io/badger/issues/869
# Data stream entries

## streams

Get references to all streams

## streams@<stream-name>

Get all availble sub keys like schema, configuration

## streams@<stream-name>/schema

Get the json schema of the stream

## streams@<stream-name>/configuration

Get the configuration of the stream

## Streams index validation

Validate all incoming "index" requests against JSON schema
