package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadSchema(t *testing.T) {
	file := filepath.Join("test", "schema.json")
	_, err := readSchema(file)
	if !assert.NoError(t, err, "readSchema(%s) should succeed", file) {
		return
	}
}

func readSchema(f string) (*Schema, error) {
	in, err := os.Open(f)
	if err != nil {
		return nil, err
	}
	return Read(in)
}

func TestValidate(t *testing.T) {
	tests := []string{
		"allof",
		"anyof",
		"business",
		"integer",
		"not",
		"numrange",
		"objectpatterns",
		"objectpropsize",
		"objectproprequired",
		"oneof",
		"strlen",
		"strpattern",
	}
	for _, name := range tests {
		schemaf := filepath.Join("test", name+".json")
		schema, err := readSchema(schemaf)
		if !assert.NoError(t, err, "reading schema file %s should succeed", schemaf) {
			return
		}

		pat := filepath.Join("test", fmt.Sprintf("%s_pass*.json", name))
		files, _ := filepath.Glob(pat)
		for _, passf := range files {
			t.Logf("Testing schema against %s", passf)
			passin, err := os.Open(passf)
			if !assert.NoError(t, err, "os.Open(%s) should succeed", passf) {
				return
			}
			var m map[string]interface{} // XXX should test against structs
			if !assert.NoError(t, json.NewDecoder(passin).Decode(&m), "json.Decode should succeed") {
				return
			}

			if !assert.NoError(t, schema.Validate(m), "schema.Validate should succeed") {
				return
			}
		}

		pat = filepath.Join("test", fmt.Sprintf("%s_fail*.json", name))
		files, _ = filepath.Glob(pat)
		for _, failf := range files {
			t.Logf("Testing schema against %s", failf)
			failin, err := os.Open(failf)
			if !assert.NoError(t, err, "os.Open(%s) should succeed", failf) {
				return
			}
			var m map[string]interface{} // XXX should test against structs
			if !assert.NoError(t, json.NewDecoder(failin).Decode(&m), "json.Decode should succeed") {
				return
			}

			if !assert.Error(t, schema.Validate(m), "schema.Validate should fail") {
				return
			}
		}
	}
}
