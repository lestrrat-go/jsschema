# go-jsschema

[![Build Status](https://travis-ci.org/lestrrat/go-jsschema.svg?branch=master)](https://travis-ci.org/lestrrat/go-jsschema)

[![GoDoc](https://godoc.org/github.com/lestrrat/go-jsschema?status.svg)](https://godoc.org/github.com/lestrrat/go-jsschema)

JSON Schema for Go

# CAVEATS

* Dependencies: Currently schema dependencies are NOT supported. If you must specify a schema, you probably should define a non-required property (PRs welcome). See [go-jsval](https://github.com/lestrrat/go-jsval) for a validator that handles it.

# SYNOPSIS

```go
package schema_test

import (
  "log"

  "github.com/lestrrat/go-jsschema"
)

func Example() {
  s, err := schema.ReadFile("schema.json")
  if err != nil {
    log.Printf("failed to read schema: %s", err)
    return
  }

  for name, pdef := range s.Properties {
    // Do what you will with `pdef`, which contain
    // Schema information for `name` property
    _ = name
    _ = pdef
  }

  // You can also validate an arbitrary piece of data
  var p interface{} // initialize using json.Unmarshal...
  if err := s.Validate(p); err != nil {
    log.Printf("failed to validate data: %s", err)
  }
}
```

# TODO

* Properly resolve ids and $refs (it works in simple cases, but elaborate scopes probably don't work)
* Would be nice to swap `schema.Validate` with `jsval.Validate`

# References

| Name                                                     | Notes                         |
|:--------------------------------------------------------:|:------------------------------|
| [go-jsval](https://github.com/lestrrat/go-jsval)         | Validator generator           |
| [go-jsref](https://github.com/lestrrat/go-jsref)         | JSON Reference implementation |
| [go-jspointer](https://github.com/lestrrat/go-jspointer) | JSON Pointer implementations  |
