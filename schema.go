package schema

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"net"
	"net/mail"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/lestrrat/go-jsref"
	"github.com/lestrrat/go-jsref/provider"
	"github.com/lestrrat/go-pdebug"
	"github.com/lestrrat/go-structinfo"
)

// This is used to check against result of reflect.MapIndex
var zeroval = reflect.Value{}
var _schema *Schema

func init() {
	buildJSSchema()
}

func New() *Schema {
	resolver := jsref.New()

	mp := provider.NewMap()
	mp.Set(SchemaURL, _schema)
	resolver.AddProvider(mp)

	s := &Schema{
		schemaByID: make(map[string]*Schema),
		resolver:   resolver,
	}
	return s
}

func ReadFile(f string) (*Schema, error) {
	in, err := os.Open(f)
	if err != nil {
		return nil, err
	}
	defer in.Close()
	return Read(in)
}

func Read(in io.Reader) (*Schema, error) {
	s := New()
	dec := json.NewDecoder(in)
	if err := dec.Decode(s); err != nil {
		return nil, err
	}

	s.applyParentSchema()
	return s, nil
}

func (s *Schema) setParent(v *Schema) {
	s.parent = v
}

func (s *Schema) applyParentSchema() {
	// Find all components that may be a Schema
	for _, v := range s.Definitions {
		v.setParent(s)
		v.applyParentSchema()
	}

	if props := s.AdditionalProperties; props != nil {
		if sc := props.Schema; sc != nil {
			sc.setParent(s)
			sc.applyParentSchema()
		}
	}
	if items := s.AdditionalItems; items != nil {
		if sc := items.Schema; sc != nil {
			sc.setParent(s)
			sc.applyParentSchema()
		}
	}
	if items := s.Items; items != nil {
		for _, v := range items.Schemas {
			v.setParent(s)
			v.applyParentSchema()
		}
	}

	for _, v := range s.Properties {
		v.setParent(s)
		v.applyParentSchema()
	}

	for _, v := range s.AllOf {
		v.setParent(s)
		v.applyParentSchema()
	}

	for _, v := range s.AnyOf {
		v.setParent(s)
		v.applyParentSchema()
	}

	for _, v := range s.OneOf {
		v.setParent(s)
		v.applyParentSchema()
	}

	if v := s.Not; v != nil {
		v.setParent(s)
		v.applyParentSchema()
	}
}

func (s Schema) BaseURL() *url.URL {
	scope := s.Scope()
	u, err := url.Parse(scope)
	if err != nil {
		// XXX hmm, not sure what to do here
		u = &url.URL{}
	}

	return u
}

func (s *Schema) Root() *Schema {
	if s.parent == nil {
		if pdebug.Enabled {
			pdebug.Printf("Schema %p is root", s)
		}
		return s
	}

	return s.parent.Root()
}

func (s *Schema) findSchemaByID(id string) (*Schema, error) {
	if s.ID == id {
		return s, nil
	}

	// XXX Quite unimplemented
	return nil, ErrSchemaNotFound
}

func (s *Schema) ResolveID(id string) (r *Schema, err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START Schema.ResolveID '%s'", id)
		defer func() {
			if err != nil {
				g.IRelease("END Schema.ResolveID '%s': error %s", id, err)
			} else {
				g.IRelease("END Schema.ResolveID '%s' -> %p", id, r)
			}
		}()
	}
	root := s.Root()

	var ok bool
	r, ok = root.schemaByID[id]
	if ok {
		return
	}

	r, err = root.findSchemaByID(id)
	if err != nil {
		return
	}

	root.schemaByID[id] = r
	return
}

func (s Schema) ResolveURL(v string) (u *url.URL, err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START Schema.ResolveURL '%s'", v)
		defer func() {
			if err != nil {
				g.IRelease("END Schema.ResolveURL '%s': error %s", v, err)
			} else {
				g.IRelease("END Schema.ResolveURL '%s' -> '%s'", v, u)
			}
		}()
	}
	base := s.BaseURL()
	if pdebug.Enabled {
		pdebug.Printf("Using base URL '%s'", base)
	}
	u, err = base.Parse(v)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// Resolve the current schema reference, if '$ref' exists
func resolveSchemaReference(s *Schema) (res *Schema, err error) {
	if s.Reference == "" {
		return s, nil
	}

	if pdebug.Enabled {
		g := pdebug.IPrintf("START resolveSchemaReference (%s)", s.Reference)
		defer func() {
			if err != nil {
				g.IRelease("END resolveSchemaReference (%s): %s", s.Reference, err)
			} else {
				g.IRelease("END resolveSchemaReference (%s)", s.Reference)
			}
		}()
	}

	thing, err := s.resolver.Resolve(s.Root(), s.Reference)
	if err != nil {
		return nil, ErrInvalidReference{Reference: s.Reference, Message: err.Error()}
	}

	ref, ok := thing.(*Schema)
	if !ok {
		return nil, ErrInvalidReference{Reference: s.Reference, Message: "returned element is not a Schema"}
	}

	return ref, nil
}

func (s Schema) Validate(v interface{}) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START Schema.Validate")
		defer g.IRelease("END Schema.Validate")

		buf, _ := json.MarshalIndent(s, "", "  ")
		pdebug.Printf("schema to validate against: %s", buf)
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	if err := validate(rv, &s); err != nil {
		return err
	}

	return nil
}

func (s Schema) isPropRequired(pname string) bool {
	for _, name := range s.Required {
		if name == pname {
			return true
		}
	}
	return false
}

// getProps return all of the property names for this object.
// XXX Map keys can be something other than strings, but
// we can't really allow it?
func getPropNames(rv reflect.Value) ([]string, error) {
	var keys []string
	switch rv.Kind() {
	case reflect.Map:
		vk := rv.MapKeys()
		keys = make([]string, len(vk))
		for i, v := range vk {
			if v.Kind() != reflect.String {
				return nil, errors.New("panic: can only handle maps with string keys")
			}
			keys[i] = v.String()
		}
	case reflect.Struct:
		if keys = structinfo.JSONFieldsFromStruct(rv); keys == nil {
			// Can't happen, because we check for reflect.Struct,
			// but for completeness
			return nil, errors.New("panic: can only handle structs")
		}
	default:
		return nil, errors.New("cannot get property names from this value")
	}

	return keys, nil
}

func getProp(rv reflect.Value, pname string) reflect.Value {
	switch rv.Kind() {
	case reflect.Map:
		pv := reflect.ValueOf(pname)
		return rv.MapIndex(pv)
	case reflect.Struct:
		i := structinfo.StructFieldFromJSONName(rv, pname)
		if i < 0 {
			return zeroval
		}

		return rv.Field(i)
	default:
		return zeroval
	}
}

func matchType(t PrimitiveType, list PrimitiveTypes) (err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START matchType '%s'", t)
		defer func() {
			if err == nil {
				g.IRelease("END matchType '%s' (PASS)", t)
			} else {
				g.IRelease("END matchType '%s': error %s", t, err)
			}
		}()
	}

	if len(list) == 0 {
		return nil
	}

	for _, tp := range list {
		switch tp {
		case t:
		default:
			return ErrInvalidType
		}
	}
	if pdebug.Enabled {
		pdebug.Printf("Type match succeeded")
	}
	return nil
}

func validateProp(c reflect.Value, pname string, def *Schema, required bool) (err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START validateProp '%s'", pname)
		defer func() {
			if err == nil {
				g.IRelease("END validateProp '%s' (PASS)", pname)
			} else {
				buf, _ := json.MarshalIndent(c.Interface(), "", "  ")
				pdebug.Printf("%s", buf)
				buf, _ = json.MarshalIndent(def, "", "  ")
				pdebug.Printf("%s", buf)
				g.IRelease("END validateProp '%s': error %s", pname, err)
			}
		}()
	}

	pv := getProp(c, pname)
	if pv.Kind() == reflect.Interface {
		pv = pv.Elem()
	}

	if pv == zeroval {
		// no prop by name of pname. is this required?
		if !required {
			if pdebug.Enabled {
				pdebug.Printf("Property %s not found, but is not required", pname)
			}
		} else {
			if pdebug.Enabled {
				pdebug.Printf("Property %s is required, but not found", pname)
			}
			err = ErrRequiredField{Name: pname}
		}
		return
	}

	// It's totally ok to have use an empty schema here, because
	// we might just might be checking that the property exists
	if def == nil {
		return nil
	}

	def, err = resolveSchemaReference(def)
	if err != nil {
		return
	}
	if err = validate(pv, def); err != nil {
		return
	}
	return
}

// stolen from src/net/dnsclient.go
func isDomainName(s string) bool {
	// See RFC 1035, RFC 3696.
	if len(s) == 0 {
		return false
	}
	if len(s) > 255 {
		return false
	}

	last := byte('.')
	ok := false // Ok once we've seen a letter.
	partlen := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		default:
			return false
		case 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || c == '_':
			ok = true
			partlen++
		case '0' <= c && c <= '9':
			// fine
			partlen++
		case c == '-':
			// Byte before dash cannot be dot.
			if last == '.' {
				return false
			}
			partlen++
		case c == '.':
			// Byte before dot cannot be dot, dash.
			if last == '.' || last == '-' {
				return false
			}
			if partlen > 63 || partlen == 0 {
				return false
			}
			partlen = 0
		}
		last = c
	}
	if last == '-' || partlen > 63 {
		return false
	}

	return ok
}

func validateEnum(rv reflect.Value, enum []interface{}) error {
	iv := rv.Interface()
	for _, v := range enum {
		if iv == v {
			return nil
		}
	}

	return ErrInvalidEnum
}

// Assumes rv is a string (Kind == String)
func validateString(rv reflect.Value, def *Schema) (err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START validateString")
		defer func() {
			if err != nil {
				g.IRelease("END validateString: err = %s", err)
			} else {
				g.IRelease("END validateString (PASS)")
			}
		}()
	}

	if def.MinLength.Initialized {
		if v := def.MinLength.Val; rv.Len() < v {
			err = ErrMinLengthValidationFailed{Len: rv.Len(), MinLength: v}
			return
		}
	}

	if def.MaxLength.Initialized {
		if v := def.MaxLength.Val; rv.Len() > v {
			err = ErrMaxLengthValidationFailed{Len: rv.Len(), MaxLength: v}
			return
		}
	}

	if def.Pattern != nil {
		if !def.Pattern.MatchString(rv.String()) {
			err = ErrPatternValidationFailed{Str: rv.String(), Pattern: def.Pattern}
			return
		}
	}

	if len(def.Enum) > 0 {
		if err = validateEnum(rv, def.Enum); err != nil {
			return
		}
	}

	if def.Format != "" {
		s := rv.String()
		switch def.Format {
		case FormatDateTime:
			if _, err = time.Parse(time.RFC3339, s); err != nil {
				return
			}
		case FormatEmail:
			if _, err = mail.ParseAddress(s); err != nil {
				return
			}
		case FormatHostname:
			if !isDomainName(s) {
				err = ErrInvalidHostname
				return
			}
		case FormatIPv4:
			// Should only contain numbers and "."
			for _, r := range s {
				switch {
				case r == 0x2E || 0x30 <= r && r <= 0x39:
				default:
					err = ErrInvalidIPv4
					return
				}
			}
			if addr := net.ParseIP(s); addr == nil {
				err = ErrInvalidIPv4
			}
		case FormatIPv6:
			// Should only contain numbers and ":"
			for _, r := range s {
				switch {
				case r == 0x3A || 0x30 <= r && r <= 0x39:
				default:
					err = ErrInvalidIPv6
					return
				}
			}
			if addr := net.ParseIP(s); addr == nil {
				err = ErrInvalidIPv6
			}
		case FormatURI:
			if _, err = url.Parse(s); err != nil {
				return
			}
		default:
			err = ErrInvalidFormat
			return
		}
	}

	return nil
}

func validateNumber(rv reflect.Value, def *Schema) (err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START validateNumber")
		defer func() {
			if err != nil {
				g.IRelease("END validateNumber: err = %s", err)
			} else {
				g.IRelease("END validateNumber (PASS)")
			}
		}()
	}

	if len(def.Enum) > 0 {
		if err = validateEnum(rv, def.Enum); err != nil {
			return
		}
	}

	var f float64
	// Force value to be float64 so that it's easier to handle
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		f = float64(rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		f = float64(rv.Uint())
	case reflect.Float32, reflect.Float64:
		f = rv.Float()
	}

	if def.Minimum.Initialized {
		if def.ExclusiveMinimum.Bool() {
			if f < def.Minimum.Val {
				err = ErrMinimumValidationFailed{Num: f, Min: def.Minimum.Val, Exclusive: true}
				return
			}
		} else {
			if f <= def.Minimum.Val {
				err = ErrMinimumValidationFailed{Num: f, Min: def.Minimum.Val, Exclusive: false}
				return
			}
		}
	}

	if def.Maximum.Initialized {
		if def.ExclusiveMaximum.Bool() {
			if f > def.Maximum.Val {
				err = ErrMaximumValidationFailed{Num: f, Max: def.Maximum.Val, Exclusive: true}
				return
			}
		} else {
			if f >= def.Maximum.Val {
				err = ErrMaximumValidationFailed{Num: f, Max: def.Maximum.Val, Exclusive: false}
				return
			}
		}
	}

	if v := def.MultipleOf.Val; v != 0 {
		var mod float64
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			mod = math.Mod(f, def.MultipleOf.Val)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			mod = math.Mod(f, def.MultipleOf.Val)
		case reflect.Float32, reflect.Float64:
			mod = math.Mod(f, def.MultipleOf.Val)
		}
		if mod != 0 {
			err = ErrMultipleOfValidationFailed
			return
		}
	}
	return nil
}

func validateArray(rv reflect.Value, def *Schema) (err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START validateArray")
		defer func() {
			if err != nil {
				g.IRelease("END validateArray: err = %s", err)
			} else {
				g.IRelease("END validateArray (PASS)")
			}
		}()
	}

	if def.MinItems.Initialized || def.MaxItems.Initialized {
		l := rv.Len()
		if def.MinItems.Initialized {
			if min := def.MinItems.Val; min > l {
				return ErrMinItemsValidationFailed{Len: l, MinItems: min}
			}
		}
		if def.MaxItems.Initialized {
			if max := def.MaxItems.Val; max < l {
				return ErrMaxItemsValidationFailed{Len: l, MaxItems: max}
			}
		}
	}

	if items := def.Items; items != nil {
		if items.TupleMode {
			itemLen := len(items.Schemas)
			for i := 0; i < rv.Len(); i++ {
				if pdebug.Enabled {
					pdebug.Printf("Validating element %d", i)
				}
				if i >= itemLen {
					if def.AdditionalItems == nil { // additional items not allowed
						return ErrArrayItemValidationFailed
					}
					return nil
				}
				ev := rv.Index(i)
				if ev.Kind() == reflect.Interface {
					ev = ev.Elem()
				}
				if err := validate(ev, items.Schemas[i]); err != nil {
					return err
				}
			}
		} else {
			for i := 0; i < rv.Len(); i++ {
				if pdebug.Enabled {
					pdebug.Printf("Validating element %d", i)
				}
				ev := rv.Index(i)
				if ev.Kind() == reflect.Interface {
					ev = ev.Elem()
				}
				if err := validate(ev, items.Schemas[0]); err != nil {
					return err
				}
			}
		}
	}

	if def.UniqueItems.Bool() {
		for i := 0; i < rv.Len()-1; i++ {
			ev1 := rv.Index(i).Interface()
			for j := i + 1; j < rv.Len(); j++ {
				ev2 := rv.Index(j).Interface()
				if ev1 == ev2 {
					return ErrUniqueItemsValidationFailed
				}
			}
		}
	}
	return nil
}

func validateObject(rv reflect.Value, def *Schema) error {
	names, err := getPropNames(rv)
	if err != nil {
		return err
	}

	if def.MinProperties.Initialized || def.MaxProperties.Initialized {
		// Need to count...
		count := 0
		for _, name := range names {
			if pv := getProp(rv, name); pv != zeroval {
				count++
			}
		}
		if def.MinProperties.Initialized {
			if v := def.MinProperties.Val; v > count {
				return ErrMinPropertiesValidationFailed{Num: count, Min: v}
			}
		}
		if def.MaxProperties.Initialized {
			if v := def.MaxProperties.Val; v < count {
				return ErrMaxPropertiesValidationFailed{Num: count, Max: v}
			}
		}
	}

	// Make it into a map so we don't check it multiple times
	namesMap := make(map[string]struct{})
	for _, name := range names {
		namesMap[name] = struct{}{}
	}

	for pname, pdef := range def.Properties {
		delete(namesMap, pname)
		if err := validateProp(rv, pname, pdef, def.isPropRequired(pname)); err != nil {
			return err
		}
	}

	if pp := def.PatternProperties; len(pp) > 0 {
		for pname := range namesMap {
			for pat, pdef := range pp {
				if pat.MatchString(pname) {
					delete(namesMap, pname)
					if err := validateProp(rv, pname, pdef, def.isPropRequired(pname)); err != nil {
						return err
					}
				}
			}
		}
	}

	for pname, pdef := range def.Dependencies {
		pv := getProp(rv, pname)
		if pv == zeroval {
			continue
		}

		if pdebug.Enabled {
			pdebug.Printf("Property %s has dependencies!", pname)
		}

		delete(namesMap, pname)
		switch pdef.(type) {
		case []interface{}:
			for _, depname := range pdef.([]interface{}) {
				switch depname.(type) {
				case string:
				default:
					return ErrInvalidFieldValue{Name: pname}
				}
				if err := validateProp(rv, depname.(string), nil, true); err != nil {
					return err
				}
			}
		case map[string]interface{}:
			for depname, depdef := range pdef.(map[string]*Schema) {
				if err := validateProp(rv, depname, depdef, true); err != nil {
					return err
				}
			}
		default:
			return errors.New("invalid dependency type")
		}
	}

	if def.AdditionalProperties == nil {
		if len(namesMap) > 0 {
			return ErrAdditionalProperties
		}
	} else {
		for pname := range namesMap {
			if err := validateProp(rv, pname, def.AdditionalProperties.Schema, false); err != nil {
				return err
			}
		}
	}

	return nil
}

func validate(rv reflect.Value, def *Schema) (err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START validate")
		defer func() {
			if err != nil {
				g.IRelease("END validate: err = %s", err)
			} else {
				g.IRelease("END validate (PASS)")
			}
		}()
	}

	def, err = resolveSchemaReference(def)
	if err != nil {
		return
	}

	switch {
	case def.Not != nil:
		if pdebug.Enabled {
			pdebug.Printf("Checking 'not' constraint")
		}

		// Everything is peachy, if errors do occur
		if err2 := validate(rv, def.Not); err2 == nil {
			err = ErrNotValidationFailed
			return
		}
	case len(def.AllOf) > 0:
		if pdebug.Enabled {
			pdebug.Printf("Checking 'allOf' constraint")
		}
		for _, s1 := range def.AllOf {
			if err = validate(rv, s1); err != nil {
				return
			}
		}
	case len(def.AnyOf) > 0:
		if pdebug.Enabled {
			pdebug.Printf("Checking 'anyOf' constraint")
		}
		ok := false
		for _, s1 := range def.AnyOf {
			// don't use err from upper scope
			if err := validate(rv, s1); err == nil {
				ok = true
				break
			}
		}
		if !ok {
			err = ErrAnyOfValidationFailed
			return
		}
	case len(def.OneOf) > 0:
		if pdebug.Enabled {
			pdebug.Printf("Checking 'oneOf' constraint")
		}
		count := 0
		for _, s1 := range def.OneOf {
			// don't use err from upper scope
			if err := validate(rv, s1); err == nil {
				count++
			}
		}
		if count != 1 {
			err = ErrOneOfValidationFailed
			return
		}
	}

	switch rv.Kind() {
	case reflect.Map, reflect.Struct:
		if err = matchType(ObjectType, def.Type); err != nil {
			return
		}
		if err = validateObject(rv, def); err != nil {
			return
		}
	case reflect.Slice:
		if err = matchType(ArrayType, def.Type); err != nil {
			return
		}

		if err = validateArray(rv, def); err != nil {
			return
		}
	case reflect.Bool:
		if err = matchType(BooleanType, def.Type); err != nil {
			return
		}
	case reflect.String:
		// Make sure string type is allowed here
		if err = matchType(StringType, def.Type); err != nil {
			return
		}
		if err = validateString(rv, def); err != nil {
			return
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64:
		typeOK := false
		intOK := true
		if err = matchType(IntegerType, def.Type); err == nil {
			// Check if this is a valid integer
			if f := rv.Float(); math.Floor(f) == f {
				// it's valid, bail out of this type checking, because we're all good
				typeOK = true
				goto TYPECHECK_DONE
			}
			intOK = false
		}

		if err = matchType(NumberType, def.Type); err != nil {
			return
		}
		typeOK = true
	TYPECHECK_DONE:
		if !typeOK {
			if !intOK {
				err = ErrIntegerValidationFailed
			} else {
				err = ErrNumberValidationFailed
			}
			return
		}

		if err = validateNumber(rv, def); err != nil {
			return
		}
	default:
		if pdebug.Enabled {
			pdebug.Printf("object type is invalid: %s", rv.Kind())
		}
		err = ErrInvalidType
		return
	}
	return nil
}

func (s Schema) Scope() string {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START Schema.Scope")
		defer g.IRelease("END Schema.Scope")
	}
	if s.ID != "" || s.parent == nil {
		if pdebug.Enabled {
			pdebug.Printf("Returning id '%s'", s.ID)
		}
		return s.ID
	}

	return s.parent.Scope()
}

func buildJSSchema() {
	const src = `{
  "id": "http://json-schema.org/draft-04/schema#",
  "$schema": "http://json-schema.org/draft-04/schema#",
  "description": "Core schema meta-schema",
  "definitions": {
    "schemaArray": {
      "type": "array",
      "minItems": 1,
      "items": { "$ref": "#" }
    },
    "positiveInteger": {
      "type": "integer",
      "minimum": 0
    },
    "positiveIntegerDefault0": {
      "allOf": [ { "$ref": "#/definitions/positiveInteger" }, { "default": 0 } ]
    },
    "simpleTypes": {
      "enum": [ "array", "boolean", "integer", "null", "number", "object", "string" ]
    },
    "stringArray": {
      "type": "array",
      "items": { "type": "string" },
      "minItems": 1,
      "uniqueItems": true
    }
  },
  "type": "object",
  "properties": {
    "id": {
      "type": "string",
      "format": "uri"
    },
    "$schema": {
      "type": "string",
      "format": "uri"
    },
    "title": {
      "type": "string"
    },
    "description": {
      "type": "string"
    },
    "default": {},
    "multipleOf": {
      "type": "number",
      "minimum": 0,
      "exclusiveMinimum": true
    },
    "maximum": {
      "type": "number"
    },
    "exclusiveMaximum": {
      "type": "boolean",
      "default": false
    },
    "minimum": {
      "type": "number"
    },
    "exclusiveMinimum": {
      "type": "boolean",
      "default": false
    },
    "maxLength": { "$ref": "#/definitions/positiveInteger" },
    "minLength": { "$ref": "#/definitions/positiveIntegerDefault0" },
    "pattern": {
      "type": "string",
      "format": "regex"
    },
    "additionalItems": {
      "anyOf": [
        { "type": "boolean" },
        { "$ref": "#" }
      ],
      "default": {}
    },
    "items": {
      "anyOf": [
        { "$ref": "#" },
        { "$ref": "#/definitions/schemaArray" }
      ],
      "default": {}
    },
    "maxItems": { "$ref": "#/definitions/positiveInteger" },
    "minItems": { "$ref": "#/definitions/positiveIntegerDefault0" },
    "uniqueItems": {
      "type": "boolean",
      "default": false
    },
    "maxProperties": { "$ref": "#/definitions/positiveInteger" },
    "minProperties": { "$ref": "#/definitions/positiveIntegerDefault0" },
    "required": { "$ref": "#/definitions/stringArray" },
    "additionalProperties": {
      "anyOf": [
        { "type": "boolean" },
        { "$ref": "#" }
      ],
      "default": {}
    },
    "definitions": {
      "type": "object",
      "additionalProperties": { "$ref": "#" },
      "default": {}
    },
    "properties": {
      "type": "object",
      "additionalProperties": { "$ref": "#" },
      "default": {}
    },
    "patternProperties": {
      "type": "object",
      "additionalProperties": { "$ref": "#" },
      "default": {}
    },
    "dependencies": {
      "type": "object",
      "additionalProperties": {
        "anyOf": [
          { "$ref": "#" },
          { "$ref": "#/definitions/stringArray" }
        ]
      }
    },
    "enum": {
      "type": "array",
      "minItems": 1,
      "uniqueItems": true
    },
    "type": {
      "anyOf": [
        { "$ref": "#/definitions/simpleTypes" },
        {
          "type": "array",
          "items": { "$ref": "#/definitions/simpleTypes" },
          "minItems": 1,
          "uniqueItems": true
        }
      ]
    },
    "allOf": { "$ref": "#/definitions/schemaArray" },
    "anyOf": { "$ref": "#/definitions/schemaArray" },
    "oneOf": { "$ref": "#/definitions/schemaArray" },
    "not": { "$ref": "#" }
  },
  "dependencies": {
    "exclusiveMaximum": [ "maximum" ],
    "exclusiveMinimum": [ "minimum" ]
  },
  "default": {}
}`
	s, err := Read(strings.NewReader(src))
	if err != nil {
		// We regret to inform you that if we can't parse this
		// schema, then we have a real real real problem, so we're
		// going to panic
		panic("failed to parse main JSON Schema schema: " + err.Error())
	}
	_schema = s
}
