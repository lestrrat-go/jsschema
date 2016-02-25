package validator

import (
	"sync"

	"github.com/lestrrat/go-jsschema"
	"github.com/lestrrat/go-jsval"
	"github.com/lestrrat/go-jsval/builder"
)

type Validator struct {
	lock   sync.Mutex
	schema *schema.Schema
	jsval  *jsval.JSVal
}

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

func (v *Validator) Validate(x interface{}) error {
	jsv, err := v.validator()
	if err != nil {
		return err
	}
	return jsv.Validate(x)
}
