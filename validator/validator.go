package validator

import (
	"sync"

	"github.com/lestrrat/go-jsschema"
	"github.com/lestrrat/go-jsval"
	"github.com/lestrrat/go-jsval/builder"
)

// Validator is an object that wraps jsval.Validator, and
// can be used to validate an object against a schema
type Validator struct {
	lock   sync.Mutex
	schema *schema.Schema
	jsval  *jsval.JSVal
}

// New creates a new Validator from a JSON Schema
func New(s *schema.Schema) *Validator {
	return &Validator{
		schema: s,
	}
}

func (v *Validator) validator() (*jsval.JSVal, error) {
	v.lock.Lock()
	defer v.lock.Unlock()

	if v.jsval == nil {
		b := builder.New()
		jsv, err := b.Build(v.schema)
		if err != nil {
			return nil, err
		}
		v.jsval = jsv
	}
	return v.jsval, nil
}

// Validate takes an arbitrary piece of data and
// validates it against the schema.
func (v *Validator) Validate(x interface{}) error {
	jsv, err := v.validator()
	if err != nil {
		return err
	}
	return jsv.Validate(x)
}
