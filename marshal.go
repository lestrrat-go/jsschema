package schema

import (
	"encoding/json"
	"reflect"
	"regexp"

	"github.com/lestrrat/go-pdebug"
)

func extractNumber(n *Number, m map[string]interface{}, s string) error {
	v, ok := m[s]
	if !ok {
		return nil
	}

	switch v.(type) {
	case float64:
	default:
		return ErrInvalidFieldValue{Name: s}
	}

	n.Val = v.(float64)
	n.Initialized = true
	return nil
}

func extractInt(n *Integer, m map[string]interface{}, s string) error {
	v, ok := m[s]
	if !ok {
		return nil
	}

	switch v.(type) {
	case float64:
		n.Val = int(v.(float64))
		n.Initialized = true
	default:
		return ErrInvalidFieldValue{Name: s}
	}

	return nil
}

func extractBool(b *Bool, m map[string]interface{}, s string, def bool) error {
	b.Default = def
	v, ok := m[s]
	if !ok {
		return nil
	}

	switch v.(type) {
	case bool:
	default:
		return ErrInvalidFieldValue{Name: s}
	}

	b.Val = v.(bool)
	b.Initialized = true
	return nil
}

func extractString(s *string, m map[string]interface{}, name string) error {
	if v, ok := m[name]; ok {
		switch v.(type) {
		case string:
			*s = v.(string)
			return nil
		default:
			return ErrInvalidFieldValue{Name: name}
		}
	}

	return nil
}

func convertStringList(l *[]string, v interface{}) error {
	switch v.(type) {
	case string: // One element
		*l = []string{v.(string)}
	case []interface{}: // List of elements.
		src := v.([]interface{})
		*l = make([]string, len(src))
		for i, x := range src {
			switch x.(type) {
			case string:
			default:
				return ErrInvalidStringArray
			}

			(*l)[i] = x.(string)
		}
	default:
		return ErrInvalidStringArray
	}
	return nil
}

func extractStringList(l *[]string, m map[string]interface{}, s string) error {
	v, ok := m[s]
	if !ok {
		return nil
	}
	return convertStringList(l, v)
}

func extractFormat(f *Format, m map[string]interface{}, s string) error {
	var v string
	if err := extractString(&v, m, s); err != nil {
		return err
	}
	*f = Format(v)
	return nil
}

func extractJSPointer(s *string, m map[string]interface{}, name string) error {
	return extractString(s, m, name)
}

func extractInterface(r *interface{}, m map[string]interface{}, s string) error {
	if v, ok := m[s]; ok {
		*r = v
	}
	return nil
}

func extractInterfaceList(l *[]interface{}, m map[string]interface{}, s string) error {
	v, ok := m[s]
	if !ok {
		return nil
	}

	switch v.(type) {
	case []interface{}:
		src := v.([]interface{})
		*l = make([]interface{}, len(src))
		copy(*l, src)
		return nil
	default:
		return ErrInvalidFieldValue{Name: s}
	}
}

func extractRegexp(r **regexp.Regexp, m map[string]interface{}, s string) error {
	v, ok := m[s]
	if !ok {
		return nil
	}
	switch v.(type) {
	case string:
		rx, err := regexp.Compile(v.(string))
		if err != nil {
			return err
		}
		*r = rx
		return nil
	default:
		return ErrInvalidType
	}
}

func extractSchema(s **Schema, m map[string]interface{}, name string) error {
	v, ok := m[name]
	if !ok {
		return nil
	}

	if pdebug.Enabled {
		pdebug.Printf("Found property '%s'", name)
	}

	switch v.(type) {
	case map[string]interface{}:
	default:
		return ErrInvalidType
	}
	*s = New()
	if err := (*s).Extract(v.(map[string]interface{})); err != nil {
		return err
	}
	return nil
}

func (l *SchemaList) ExtractIfPresent(m map[string]interface{}, name string) error {
	v, ok := m[name]
	if !ok {
		return nil
	}

	if pdebug.Enabled {
		pdebug.Printf("Found property '%s'", name)
	}

	return l.Extract(v)
}

func (l *SchemaList) Extract(v interface{}) error {
	switch v.(type) {
	case []interface{}:
		src := v.([]interface{})
		*l = make([]*Schema, len(src))
		for i, d := range src {
			s := New()
			if err := s.Extract(d.(map[string]interface{})); err != nil {
				return err
			}
			(*l)[i] = s
		}
		return nil
	case map[string]interface{}:
		s := New()
		if err := s.Extract(v.(map[string]interface{})); err != nil {
			return err
		}
		*l = []*Schema{s}
		return nil
	default:
		return ErrInvalidSchemaList
	}
}

func extractSchemaMapEntry(s *Schema, name string, m map[string]interface{}) error {
	if pdebug.Enabled {
		g := pdebug.Marker("Schema map entry '%s'", name)
		defer g.End()
	}
	return s.Extract(m)
}

func extractSchemaMap(m map[string]interface{}, name string) (map[string]*Schema, error) {
	v, ok := m[name]
	if !ok {
		return nil, nil
	}

	switch v.(type) {
	case map[string]interface{}:
	default:
		return nil, ErrInvalidFieldValue{Name: name}
	}

	r := make(map[string]*Schema)
	for k, data := range v.(map[string]interface{}) {
		// data better be a map
		switch data.(type) {
		case map[string]interface{}:
		default:
			return nil, ErrInvalidFieldValue{Name: name}
		}

		s := New()
		if err := extractSchemaMapEntry(s, k, data.(map[string]interface{})); err != nil {
			return nil, err
		}
		r[k] = s

		if k == "domain" {
			if pdebug.Enabled {
				pdebug.Printf("after extractSchemaMapEntry: %#v", s.Extras)
			}
		}
	}
	return r, nil
}

func extractRegexpToSchemaMap(m map[string]interface{}, name string) (map[*regexp.Regexp]*Schema, error) {
	if v, ok := m[name]; ok {
		switch v.(type) {
		case map[string]interface{}:
		default:
			return nil, ErrInvalidFieldValue{Name: name}
		}

		r := make(map[*regexp.Regexp]*Schema)
		for k, data := range v.(map[string]interface{}) {
			// data better be a map
			switch data.(type) {
			case map[string]interface{}:
			default:
				return nil, ErrInvalidFieldValue{Name: name}
			}
			s := New()
			if err := s.Extract(data.(map[string]interface{})); err != nil {
				return nil, err
			}

			rx, err := regexp.Compile(k)
			if err != nil {
				return nil, err
			}

			r[rx] = s
		}
		return r, nil
	}
	return nil, nil
}

func extractItems(res **ItemSpec, m map[string]interface{}, name string) error {
	v, ok := m[name]
	if !ok {
		return nil
	}

	if pdebug.Enabled {
		pdebug.Printf("Found array element '%s'", name)
	}

	tupleMode := false
	switch v.(type) {
	case []interface{}:
		tupleMode = true
	case map[string]interface{}:
	default:
		return ErrInvalidFieldValue{Name: name}
	}

	items := ItemSpec{}
	items.TupleMode = tupleMode

	var err error

	if err = items.Schemas.ExtractIfPresent(m, name); err != nil {
		return err
	}
	*res = &items
	return nil
}

func extractDependecies(res *DependencyMap, m map[string]interface{}, name string) error {
	v, ok := m[name]
	if !ok {
		return nil
	}

	switch v.(type) {
	case map[string]interface{}:
	default:
		return ErrInvalidFieldValue{Name: name}
	}

	m = v.(map[string]interface{})
	if len(m) == 0 {
		return nil
	}

	return res.extract(m)
}

func (dm *DependencyMap) extract(m map[string]interface{}) error {
	dm.Names = make(map[string][]string)
	dm.Schemas = make(map[string]*Schema)
	for k, p := range m {
		switch p.(type) {
		case []interface{}:
			// This list needs to be a list of strings
			var l []string
			if err := convertStringList(&l, p.([]interface{})); err != nil {
				return err
			}

			dm.Names[k] = l
		case map[string]interface{}:
			s := New()
			if err := s.Extract(p.(map[string]interface{})); err != nil {
				return err
			}
			dm.Schemas[k] = s
		}
	}

	return nil
}

func (s *Schema) UnmarshalJSON(data []byte) error {
	m := map[string]interface{}{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	return s.Extract(m)
}

func (s *Schema) Extract(m map[string]interface{}) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START Schema.Extract")
		defer g.IRelease("END Schema.Extract")
	}

	var err error

	if err = extractString(&s.ID, m, "id"); err != nil {
		return ErrExtract{Field: "id", Err: err}
	}

	if err = extractString(&s.Title, m, "title"); err != nil {
		return ErrExtract{Field: "title", Err: err}
	}

	if err = extractString(&s.Description, m, "description"); err != nil {
		return ErrExtract{Field: "description", Err: err}
	}

	if err = extractStringList(&s.Required, m, "required"); err != nil {
		return ErrExtract{Field: "required", Err: err}
	}

	if err = extractJSPointer(&s.SchemaRef, m, "$schema"); err != nil {
		return ErrExtract{Field: "$schema", Err: err}
	}

	if err = extractJSPointer(&s.Reference, m, "$ref"); err != nil {
		return ErrExtract{Field: "$ref", Err: err}
	}

	if err = extractFormat(&s.Format, m, "format"); err != nil {
		return ErrExtract{Field: "format", Err: err}
	}

	if err = extractInterfaceList(&s.Enum, m, "enum"); err != nil {
		return ErrExtract{Field: "enum", Err: err}
	}

	if err = extractInterface(&s.Default, m, "default"); err != nil {
		return ErrExtract{Field: "default", Err: err}
	}

	if v, ok := m["type"]; ok {
		switch v.(type) {
		case string:
			t, err := primitiveFromString(v.(string))
			if err != nil {
				return ErrExtract{Field: "type", Err: err}
			}
			s.Type = PrimitiveTypes{t}
		case []interface{}:
			l := v.([]interface{})
			s.Type = make(PrimitiveTypes, len(l))
			for i, ts := range l {
				switch ts.(type) {
				case string:
				default:
					return ErrExtract{
						Field: "type",
						Err: ErrInvalidFieldValue{
							Name: "type",
							Kind: reflect.ValueOf(ts).Kind().String(),
						},
					}
				}
				t, err := primitiveFromString(ts.(string))
				if err != nil {
					return err
				}
				s.Type[i] = t
			}
		default:
			return ErrExtract{Field: "type", Err: ErrInvalidFieldValue{Name: "type", Kind: reflect.ValueOf(v).Kind().String()}}
		}
	}

	if s.Definitions, err = extractSchemaMap(m, "definitions"); err != nil {
		return ErrExtract{Field: "definitions", Err: err}
	}

	if err = extractItems(&s.Items, m, "items"); err != nil {
		return ErrExtract{Field: "items", Err: err}
	}

	if err = extractRegexp(&s.Pattern, m, "pattern"); err != nil {
		return ErrExtract{Field: "pattern", Err: err}
	}

	if extractInt(&s.MinLength, m, "minLength"); err != nil {
		return ErrExtract{Field: "minLength", Err: err}
	}

	if extractInt(&s.MaxLength, m, "maxLength"); err != nil {
		return ErrExtract{Field: "maxLength", Err: err}
	}

	if extractInt(&s.MinItems, m, "minItems"); err != nil {
		return ErrExtract{Field: "minItems", Err: err}
	}

	if extractInt(&s.MaxItems, m, "maxItems"); err != nil {
		return ErrExtract{Field: "maxItems", Err: err}
	}

	if err = extractBool(&s.UniqueItems, m, "uniqueItems", false); err != nil {
		return ErrExtract{Field: "uniqueItems", Err: err}
	}

	if err = extractInt(&s.MaxProperties, m, "maxProperties"); err != nil {
		return ErrExtract{Field: "maxProperties", Err: err}
	}

	if err = extractInt(&s.MinProperties, m, "minProperties"); err != nil {
		return ErrExtract{Field: "minProperties", Err: err}
	}

	if err = extractNumber(&s.Minimum, m, "minimum"); err != nil {
		return ErrExtract{Field: "minimum", Err: err}
	}

	if err = extractBool(&s.ExclusiveMinimum, m, "exclusiveMinimum", false); err != nil {
		return ErrExtract{Field: "exclusiveMinimum", Err: err}
	}

	if err = extractNumber(&s.Maximum, m, "maximum"); err != nil {
		return ErrExtract{Field: "maximum", Err: err}
	}

	if err = extractBool(&s.ExclusiveMaximum, m, "exclusiveMaximum", false); err != nil {
		return ErrExtract{Field: "exclusiveMaximum", Err: err}
	}

	if err = extractNumber(&s.MultipleOf, m, "multipleOf"); err != nil {
		return ErrExtract{Field: "multipleOf", Err: err}
	}

	if s.Properties, err = extractSchemaMap(m, "properties"); err != nil {
		return ErrExtract{Field: "properties", Err: err}
	}

	if err = extractDependecies(&s.Dependencies, m, "dependencies"); err != nil {
		return ErrExtract{Field: "dependencies", Err: err}
	}

	if _, ok := m["additionalItems"]; !ok {
		// doesn't exist. it's an empty schema
		s.AdditionalItems = &AdditionalItems{}
	} else {
		var b Bool
		if err = extractBool(&b, m, "additionalItems", true); err == nil {
			if b.Bool() {
				s.AdditionalItems = &AdditionalItems{}
			}
		} else {
			// Oh, it's not a boolean?
			var apSchema *Schema
			if err = extractSchema(&apSchema, m, "additionalItems"); err != nil {
				return ErrExtract{Field: "additionalItems", Err: err}
			}
			s.AdditionalItems = &AdditionalItems{apSchema}
		}
	}

	if _, ok := m["additionalProperties"]; !ok {
		// doesn't exist. it's an empty schema
		s.AdditionalProperties = &AdditionalProperties{}
	} else {
		var b Bool
		if err = extractBool(&b, m, "additionalProperties", true); err == nil {
			if b.Bool() {
				s.AdditionalProperties = &AdditionalProperties{}
			}
		} else {
			// Oh, it's not a boolean?
			var apSchema *Schema
			if err = extractSchema(&apSchema, m, "additionalProperties"); err != nil {
				return ErrExtract{Field: "additionalProperties", Err: err}
			}
			s.AdditionalProperties = &AdditionalProperties{apSchema}
		}
	}

	if s.PatternProperties, err = extractRegexpToSchemaMap(m, "patternProperties"); err != nil {
		return ErrExtract{Field: "patternProperties", Err: err}
	}

	if err = s.AllOf.ExtractIfPresent(m, "allOf"); err != nil {
		return ErrExtract{Field: "allOf", Err: err}
	}

	if err = s.AnyOf.ExtractIfPresent(m, "anyOf"); err != nil {
		return ErrExtract{Field: "anyOf", Err: err}
	}

	if err = s.OneOf.ExtractIfPresent(m, "oneOf"); err != nil {
		return ErrExtract{Field: "oneOf", Err: err}
	}

	if err = extractSchema(&s.Not, m, "not"); err != nil {
		return ErrExtract{Field: "not", Err: err}
	}

	s.applyParentSchema()

	s.Extras = make(map[string]interface{})
	for k, v := range m {
		switch k {
		case "id", "title", "description", "required", "$schema", "$ref", "format", "enum", "default", "type", "definitions", "items", "pattern", "minLength", "maxLength", "minItems", "maxItems", "uniqueItems", "maxProperties", "minProperties", "minimum", "exclusiveMinimum", "maximum", "exclusiveMaximum", "multipleOf", "properties", "dependencies", "additionalItems", "additionalProperties", "patternProperties", "allOf", "anyOf", "oneOf", "not":
			continue
		}
		if pdebug.Enabled {
			pdebug.Printf("Extracting extra field '%s'", k)
		}
		s.Extras[k] = v
	}

	if pdebug.Enabled {
		pdebug.Printf("Successfully extracted schema")
	}

	return nil
}

func place(m map[string]interface{}, name string, v interface{}) {
	m[name] = v
}

func placeString(m map[string]interface{}, name, s string) {
	if s != "" {
		place(m, name, s)
	}
}

func placeList(m map[string]interface{}, name string, l []interface{}) {
	if len(l) > 0 {
		place(m, name, l)
	}
}
func placeSchemaList(m map[string]interface{}, name string, l []*Schema) {
	if len(l) > 0 {
		place(m, name, l)
	}
}

func placeSchemaMap(m map[string]interface{}, name string, l map[string]*Schema) {
	if len(l) > 0 {
		defs := make(map[string]*Schema)
		place(m, name, defs)

		for k, v := range l {
			defs[k] = v
		}
	}
}

func placeStringList(m map[string]interface{}, name string, l []string) {
	if len(l) > 0 {
		place(m, name, l)
	}
}

func placeBool(m map[string]interface{}, name string, value Bool) {
	place(m, name, value.Bool())
}

func placeNumber(m map[string]interface{}, name string, n Number) {
	if !n.Initialized {
		return
	}
	place(m, name, n.Val)
}

func placeInteger(m map[string]interface{}, name string, n Integer) {
	if !n.Initialized {
		return
	}
	place(m, name, n.Val)
}

func (s Schema) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})

	placeString(m, "id", s.ID)
	placeString(m, "title", s.Title)
	placeString(m, "description", s.Description)
	placeString(m, "$schema", s.SchemaRef)
	placeString(m, "$ref", s.Reference)
	placeStringList(m, "required", s.Required)
	placeList(m, "enum", s.Enum)
	switch len(s.Type) {
	case 0:
	case 1:
		m["type"] = s.Type[0]
	default:
		m["type"] = s.Type
	}

	if items := s.AdditionalItems; items != nil {
		if items.Schema != nil {
			place(m, "additionalItems", items.Schema)
		}
	} else {
		place(m, "additionalItems", false)
	}

	if rx := s.Pattern; rx != nil {
		placeString(m, "pattern", rx.String())
	}
	placeInteger(m, "maxLength", s.MaxLength)
	placeInteger(m, "minLength", s.MinLength)
	placeInteger(m, "maxItems", s.MaxItems)
	placeInteger(m, "minItems", s.MinItems)
	placeInteger(m, "maxProperties", s.MaxProperties)
	placeInteger(m, "minProperties", s.MinProperties)
	if s.UniqueItems.Initialized {
		placeBool(m, "uniqueItems", s.UniqueItems)
	}
	placeSchemaMap(m, "definitions", s.Definitions)

	if items := s.Items; items != nil {
		if items.TupleMode {
			m["items"] = s.Items.Schemas
		} else {
			m["items"] = s.Items.Schemas[0]
		}
	}

	placeSchemaMap(m, "properties", s.Properties)
	if len(s.PatternProperties) > 0 {
		rxm := make(map[string]*Schema)
		for rx, rxs := range s.PatternProperties {
			rxm[rx.String()] = rxs
		}
		placeSchemaMap(m, "patternProperties", rxm)
	}

	placeSchemaList(m, "allOf", s.AllOf)
	placeSchemaList(m, "anyOf", s.AnyOf)
	placeSchemaList(m, "oneOf", s.OneOf)

	if s.Default != nil {
		m["default"] = s.Default
	}

	placeString(m, "format", string(s.Format))
	placeNumber(m, "minimum", s.Minimum)
	if s.ExclusiveMinimum.Initialized {
		placeBool(m, "exclusiveMinimum", s.ExclusiveMinimum)
	}
	placeNumber(m, "maximum", s.Maximum)
	if s.ExclusiveMaximum.Initialized {
		placeBool(m, "exclusiveMaximum", s.ExclusiveMaximum)
	}

	if ap := s.AdditionalProperties; ap != nil {
		if ap.Schema != nil {
			place(m, "additionalProperties", ap.Schema)
		}
	} else {
		// additionalProperties: false
		placeBool(m, "additionalProperties", Bool{Val: false, Initialized: true})
	}

	if s.MultipleOf.Val != 0 {
		placeNumber(m, "multipleOf", s.MultipleOf)
	}

	if v := s.Not; v != nil {
		place(m, "not", v)
	}

	deps := map[string]interface{}{}
	if v := s.Dependencies.Schemas; v != nil {
		for pname, depschema := range v {
			deps[pname] = depschema
		}
	}
	if v := s.Dependencies.Names; v != nil {
		for pname, deplist := range v {
			deps[pname] = deplist
		}
	}

	if len(deps) > 0 {
		place(m, "dependencies", deps)
	}

	if x := s.Extras; x != nil {
		for k, v := range x {
			m[k] = v
		}
	}

	return json.Marshal(m)
}
