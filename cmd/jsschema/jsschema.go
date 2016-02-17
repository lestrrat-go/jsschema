package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/lestrrat/go-jsschema"
)

func main() {
	os.Exit(_main())
}

func usage() {
	fmt.Printf("jsschema [schema file] [target file]\n")
}

func _main() int {
	if len(os.Args) < 3 {
		usage()
		return 1
	}

	schemaf, err := os.Open(os.Args[1])
	if err != nil {
		log.Printf("failed to open schema: %s", err)
		return 1
	}
	defer schemaf.Close()

	s, err := schema.Read(schemaf)
	if err != nil {
		log.Printf("failed to read schema: %s", err)
		return 1
	}

	f, err := os.Open(os.Args[2])
	if err != nil {
		log.Printf("failed to open data: %s", err)
		return 1
	}
	defer f.Close()

	in, err := ioutil.ReadAll(f)
	if err != nil {
		log.Printf("failed to read data: %s", err)
		return 1
	}

	var v interface{}
	if err := json.Unmarshal(in, &v); err != nil {
		log.Printf("failed to decode data: %s", err)
		return 1
	}

	if err := s.Validate(v); err != nil {
		log.Printf("validation failed")
		return 1
	}

	buf, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Printf("failed to encode data: %s", err)
		return 1
	}

	os.Stdout.Write(buf)
	os.Stdout.Write([]byte{'\n'})

	return 0
}