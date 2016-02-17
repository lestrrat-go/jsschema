package schema

import (
	"encoding/json"
	"regexp"
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

func extractStringList(l *[]string, m map[string]interface{}, s string) error {
	if v, ok := m[s]; ok {
		switch v.(type) {
		case string:
			*l = []string{v.(string)}
			return nil
		case []interface{}:
			src := v.([]interface{})
			*l = make([]string, len(src))
			for i, x := range src {
				switch x.(type) {
				case string:
					(*l)[i] = x.(string)
				default:
					return ErrInvalidFieldValue{Name: s}
				}
			}
			return nil
		default:
			return ErrInvalidFieldValue{Name: s}
		}
	}

	return nil
}

func extractFormat(f *Format, m map[string]interface{}, s string) error {
	var v string
	if err := extractString(&v, m, s); err != nil {
		return  err
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
		return nil
	}
	return nil
}

func extractInterfaceList(l *[]interface{}, m map[string]interface{}, s string) error {
	if v, ok := m[s]; ok {
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

	return nil
}

func extractRegexp(r **regexp.Regexp, m map[string]interface{}, s string) error {
	if v, ok := m[s]; ok {
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
	return nil
}

func extractSchema(m map[string]interface{}, name string) (*Schema, error) {
	if v, ok := m[name]; ok {
		switch v.(type) {
		case map[string]interface{}:
		default:
			return nil, ErrInvalidType
		}
		s := New()
		if err := s.extract(v.(map[string]interface{})); err != nil {
			return nil, err
		}
		return s, nil
	}
	return nil, nil
}

func extractSchemaList(m map[string]interface{}, name string) ([]*Schema, error) {
	if v, ok := m[name]; ok {
		switch v.(type) {
		case []interface{}:
			l := v.([]interface{})
			r := make([]*Schema, len(l))
			for i, d := range l {
				s := New()
				if err := s.extract(d.(map[string]interface{})); err != nil {
					return nil, err
				}
				r[i] = s
			}
			return r, nil
		case map[string]interface{}:
			s := New()
			if err := s.extract(v.(map[string]interface{})); err != nil {
				return nil, err
			}
			return []*Schema{s}, nil
		default:
			return nil, ErrInvalidFieldValue{Name: name}
		}
	}

	return nil, nil
}

func extractSchemaMap(m map[string]interface{}, name string) (map[string]*Schema, error) {
	if v, ok := m[name]; ok {
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
			if err := s.extract(data.(map[string]interface{})); err != nil {
				return nil, err
			}
			r[k] = s
		}
		return r, nil
	}
	return nil, nil
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
			if err := s.extract(data.(map[string]interface{})); err != nil {
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
	if items.Schemas, err = extractSchemaList(m, name); err != nil {
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

	deps := DependencyMap{}
	for k, p := range m {
		switch p.(type) {
		case []interface{}:
			deps[k] = p
		case map[string]interface{}:
			r := make(map[string]*Schema)
			for k, data := range p.(map[string]interface{}) {
				// data better be a map
				switch data.(type) {
				case map[string]interface{}:
				default:
					return ErrInvalidFieldValue{Name: k}
				}
				s := New()
				if err := s.extract(data.(map[string]interface{})); err != nil {
					return err
				}
				r[k] = s
			}
			deps[k] = r
		}
	}

	*res = deps
	return nil
}

func (s *Schema) UnmarshalJSON(data []byte) error {
	m := map[string]interface{}{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	return s.extract(m)
}

func (s *Schema) extract(m map[string]interface{}) error {
	var err error

	if err = extractString(&s.ID, m, "id"); err != nil {
		return err
	}

	if err = extractString(&s.Title, m, "title"); err != nil {
		return err
	}

	if err = extractString(&s.Description, m, "description"); err != nil {
		return err
	}

	if err = extractStringList(&s.Required, m, "required"); err != nil {
		return err
	}

	if err = extractJSPointer(&s.SchemaRef, m, "$schema"); err != nil {
		return err
	}

	if err = extractJSPointer(&s.Reference, m, "$ref"); err != nil {
		return err
	}

	if err = extractFormat(&s.Format, m, "format"); err != nil {
		return err
	}

	if err = extractInterfaceList(&s.Enum, m, "enum"); err != nil {
		return err
	}

	if err = extractInterface(&s.Default, m, "default"); err != nil {
		return err
	}

	if v, ok := m["type"]; ok {
		switch v.(type) {
		case string:
			t, err := primitiveFromString(v.(string))
			if err != nil {
				return err
			}
			s.Type = PrimitiveTypes{t}
		case []string:
			l := v.([]string)
			s.Type = make(PrimitiveTypes, len(l))
			for i, ts := range l {
				t, err := primitiveFromString(ts)
				if err != nil {
					return err
				}
				s.Type[i] = t
			}
		default:
			return ErrInvalidFieldValue{Name: "type"}
		}
	}

	if s.Definitions, err = extractSchemaMap(m, "definitions"); err != nil {
		return err
	}

	if err = extractItems(&s.Items, m, "items"); err != nil {
		return err
	}

	if err = extractRegexp(&s.Pattern, m, "pattern"); err != nil {
		return err
	}

	if extractInt(&s.MinLength, m, "minLength"); err != nil {
		return err
	}

	if extractInt(&s.MaxLength, m, "maxLength"); err != nil {
		return err
	}

	if extractInt(&s.MinItems, m, "minItems"); err != nil {
		return err
	}

	if extractInt(&s.MaxItems, m, "maxItems"); err != nil {
		return err
	}

	if err = extractBool(&s.UniqueItems, m, "uniqueItems", false); err != nil {
		return err
	}

	if err = extractInt(&s.MaxProperties, m, "maxProperties"); err != nil {
		return err
	}

	if err = extractInt(&s.MinProperties, m, "minProperties"); err != nil {
		return err
	}

	if err = extractNumber(&s.Minimum, m, "minimum"); err != nil {
		return err
	}

	if err = extractBool(&s.ExclusiveMinimum, m, "exclusiveminimum", false); err != nil {
		return err
	}

	if err = extractNumber(&s.Maximum, m, "maximum"); err != nil {
		return err
	}

	if err = extractBool(&s.ExclusiveMaximum, m, "exclusivemaximum", false); err != nil {
		return err
	}

	if err = extractNumber(&s.MultipleOf, m, "multipleOf"); err != nil {
		return err
	}

	if s.Properties, err = extractSchemaMap(m, "properties"); err != nil {
		return err
	}

	if err = extractDependecies(&s.Dependencies, m, "dependencies"); err != nil {
		return err
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
			if apSchema, err = extractSchema(m, "additionalItems"); err != nil {
				return err
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
			if apSchema, err = extractSchema(m, "additionalProperties"); err != nil {
				return err
			}
			s.AdditionalProperties = &AdditionalProperties{apSchema}
		}
	}

	if s.PatternProperties, err = extractRegexpToSchemaMap(m, "patternProperties"); err != nil {
		return err
	}

	if s.Properties, err = extractSchemaMap(m, "properties"); err != nil {
		return err
	}

	if s.AllOf, err = extractSchemaList(m, "allOf"); err != nil {
		return err
	}

	if s.AnyOf, err = extractSchemaList(m, "anyOf"); err != nil {
		return err
	}

	if s.OneOf, err = extractSchemaList(m, "oneOf"); err != nil {
		return err
	}

	if s.Not, err = extractSchema(m, "not"); err != nil {
		return err
	}

	s.applyParentSchema()

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

	if v := s.Dependencies; v != nil {
		place(m, "dependencies", v)
	}

	return json.Marshal(m)
}
