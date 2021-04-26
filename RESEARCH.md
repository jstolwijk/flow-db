# Research

## Links

- https://dgraph.io/docs/badger/
- https://www.mailgun.com/blog/how-we-built-a-lucene-inspired-parser-in-go/
- http://nosqlgeek.blogspot.com/2015/11/using-key-value-store-for-full-text.html
- https://blog.kvdb.io/2019/11/01/sorting-ordered-data-in-a-key-value-store
- http://www.tiernok.com/posts/adding-index-for-a-key-value-store/
- https://blog.timescale.com/blog/time-series-compression-algorithms-explained/
- https://itnext.io/storing-time-series-in-rocksdb-a-cookbook-e873fcb117e4

## Techniques for implementing powerfull search engine

### Inverted index

https://en.wikipedia.org/wiki/Inverted_index
https://www.lighttag.io/blog/indexeddb-for-nlp/

### Partial word search

https://en.wikipedia.org/wiki/N-gram

Lets say we have a surename field in which we want to search.

We have a query to find all surenames starting with "De"

- Delvina
- Dervishi
- De Jong
- De Vries
- De Boer
- De Groot

To support this query we have to index the surename field using a n-gram where n = 2 (also known as a "bigram")

### Nearest neighbor search

https://en.wikipedia.org/wiki/Locality-sensitive_hashing

### Levenshtein distance

https://en.wikipedia.org/wiki/Levenshtein_distance

### Full text query improvements

When youre using google you get automatic corrections of your search query

https://wolfgarbe.medium.com/1000x-faster-spelling-correction-algorithm-2012-8701fcd87a5f

Implementation:
https://github.com/wolfgarbe/SymSpell

### AI

"Active learning pipeline"

https://towardsdatascience.com/how-we-built-an-ai-powered-search-engine-without-being-google-5ad93e5a8591
