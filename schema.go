package schema

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"reflect"

	"github.com/lestrrat/go-jspointer"
	"github.com/lestrrat/go-pdebug"
	"github.com/lestrrat/go-structinfo"
)

// This is used to check against result of reflect.MapIndex
var zeroval = reflect.Value{}

func New() *Schema {
	s := &Schema{
		cachedReference: make(map[string]interface{}),
		schemaByID:      make(map[string]*Schema),
	}
	return s
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

	for _, v := range s.AdditionalItems {
		v.setParent(s)
		v.applyParentSchema()
	}
	for _, v := range s.Items {
		v.setParent(s)
		v.applyParentSchema()
	}

	for _, v := range s.properties {
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
	if s.id == id {
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

func (s *Schema) ResolveReference(v string) (r interface{}, err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START Schema.ResolveReference '%s'", v)
		defer func() {
			if err != nil {
				g.IRelease("END Schema.ResolveReference '%s': error %s", v, err)
			} else {
				g.IRelease("END Schema.ResolveReference '%s'", v)
			}
		}()
	}
	u, err := s.ResolveURL(v)
	if err != nil {
		return nil, err
	}

	var ok bool
	root := s.Root()
	r, ok = root.cachedReference[u.String()]
	if ok {
		pdebug.Printf("s.ResolveReference: Cache HIT for '%s'", u)
		return
	}

	var p *jspointer.JSPointer
	p, err = jspointer.New(u.Fragment)
	if err != nil {
		return
	}

	var t *Schema
	t, err = s.ResolveID(s.Scope())
	if err != nil {
		return
	}

	r, err = p.Get(t)
	if err != nil {
		return nil, err
	}
	s.cachedReference[u.String()] = r

	if pdebug.Enabled {
		pdebug.Printf("s.ResolveReference: Resolved %s (%s)", v, u.Fragment)
	}
	return
}

func (e ErrInvalidFieldValue) Error() string {
	return fmt.Sprintf("invalid value for field %s", e.Name)
}

func (e ErrInvalidReference) Error() string {
	return fmt.Sprintf("failed to resolve reference '%s': %s", e.Reference, e.Message)
}

func (e ErrRequiredField) Error() string {
	return fmt.Sprintf("required field '%s' not found", e.Name)
}

func (s1 *Schema) merge(s2 *Schema) {
	s1.Title = s2.Title
	s1.Description = s2.Description
	s1.Default = s2.Default
	s1.Type = s2.Type
	s1.SchemaRef = s2.SchemaRef
	s1.Reference = s2.Reference
	s1.Format = s2.Format
	for k, v := range s2.Definitions {
		s1.Definitions[k] = v
	}

	// NumericValidations
	s1.MultipleOf = s2.MultipleOf
	s1.Minimum = s2.Minimum
	s1.Maximum = s2.Maximum
	s1.ExclusiveMinimum = s2.ExclusiveMinimum
	s1.ExclusiveMaximum = s1.ExclusiveMaximum

	// StringValidation
	s1.maxLength = s2.maxLength
	s1.minLength = s2.minLength
	s1.Pattern = s2.Pattern

	// ArrayValidations
	s1.AllowAdditionalItems = s2.AllowAdditionalItems
	// XXX These must be unique, but I'm going to punt it for now
	s1.AdditionalItems = append(s1.AdditionalItems, s2.AdditionalItems...)
	s1.Items = append(s1.Items, s2.Items...)
	s1.minItems = s2.minItems
	s1.maxItems = s2.maxItems
	s1.UniqueItems = s2.UniqueItems

	// ObjectValidations
	s1.MaxProperties = s2.MaxProperties
	s1.MinProperties = s2.MinProperties

	if len(s2.Required) > 0 {
		names := make(map[string]struct{})
		for _, n := range s1.Required {
			names[n] = struct{}{}
		}
		for _, n := range s2.Required {
			names[n] = struct{}{}
		}
		s1.Required = make([]string, 0, len(names))
		for n := range names {
			s1.Required = append(s1.Required, n)
		}
	}

	if len(s2.properties) > 0 {
		if s1.properties == nil {
			s1.properties = make(map[string]*Schema)
		}
		for k, v := range s2.properties {
			s1.properties[k] = v
		}
	}

	s1.AdditionalProperties = s2.AdditionalProperties
	s1.PatternProperties = s2.PatternProperties

	// XXX grr, dang it. just append for now. punt
	s1.Enum = append(s1.Enum, s2.Enum...)
	s1.AllOf = append(s1.AllOf, s2.AllOf...)
	s1.AnyOf = append(s1.AnyOf, s2.AnyOf...)
	s1.OneOf = append(s1.OneOf, s2.OneOf...)
	s1.Not = s2.Not
}

func (s *Schema) resolveAndMergeReference() error {
	if s.Reference == "" {
		return nil
	}
	ref, err := s.ResolveReference(s.Reference)
	if err != nil {
		return ErrInvalidReference{Reference: s.Reference, Message: err.Error()}
	}
	s.merge(ref.(*Schema))
	s.Reference = ""
	return nil
}

func (s Schema) Validate(v interface{}) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START Schema.Validate")
		defer g.IRelease("END Schema.Validate")
	}

	{
		buf, _ := json.MarshalIndent(s, "", "  ")
		pdebug.Printf("%s", buf)
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	if err := s.validate(rv, &s); err != nil {
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

func matchType(t PrimitiveType, list PrimitiveTypes) error {
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

func (s Schema) validateProp(c reflect.Value, pname string, def *Schema, required bool) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START Schema.validateProp '%s'", pname)
		defer g.IRelease("END Schema.validateProp '%s'", pname)
	}

	if err := def.resolveAndMergeReference(); err != nil {
		return err
	}
	pv := getProp(c, pname)
	if pv.Kind() == reflect.Interface {
		pv = pv.Elem()
	}

	if pv == zeroval {
		// no prop by name of pname. is this required?
		if required {
			if pdebug.Enabled {
				pdebug.Printf("Property %s is required, but not found", pname)
			}
			return ErrRequiredField{Name: pname}
		}
		return nil
	}

	if err := s.validate(pv, def); err != nil {
		return err
	}
	return nil
}

func (s Schema) validate(rv reflect.Value, def *Schema) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START Schema.validate")
		defer g.IRelease("END Schema.validate")
	}

	if err := def.resolveAndMergeReference(); err != nil {
		return err
	}

	switch {
	case len(def.AllOf) > 0:
		if pdebug.Enabled {
			pdebug.Printf("Checking allOf constraint")
		}
		for _, s1 := range def.AllOf {
			if err := s.validate(rv, s1); err != nil {
				return err
			}
		}
	}

	switch rv.Kind() {
	case reflect.Map, reflect.Struct:
		if err := matchType(ObjectType, def.Type); err != nil {
			return err
		}
		for pname, pdef := range def.properties {
			if err := s.validateProp(rv, pname, pdef, def.isPropRequired(pname)); err != nil {
				return err
			}
		}
	case reflect.String:
		if err := matchType(StringType, def.Type); err != nil {
			return err
		}
	default:
		return ErrInvalidType
	}
	return nil
}

func (s Schema) Scope() string {
	if s.id != "" || s.parent == nil {
		return s.id
	}

	return s.parent.Scope()
}

func (s Schema) MaxLength() int {
	return s.maxLength.Val
}

func (s Schema) MinLength() int {
	return s.minLength.Val
}

func (s Schema) MaxItems() int {
	return s.maxItems.Val
}

func (s Schema) MinItems() int {
	return s.minItems.Val
}

func (s Schema) Properties() []string {
	l := make([]string, 0, len(s.properties))
	for k := range s.properties {
		l = append(l, k)
	}
	return l
}

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

func extractInt(n *integer, m map[string]interface{}, s string) error {
	v, ok := m[s]
	if !ok {
		return nil
	}

	switch v.(type) {
	case int:
	default:
		return ErrInvalidFieldValue{Name: s}
	}

	n.Val = v.(int)
	n.Initialized = true
	return nil
}

func extractBool(b *bool, m map[string]interface{}, s string, def bool) error {
	v, ok := m[s]
	if !ok {
		*b = def
		return nil
	}

	switch v.(type) {
	case bool:
	default:
		return ErrInvalidFieldValue{Name: s}
	}

	*b = v.(bool)
	return nil
}

func extractString(m map[string]interface{}, s string) (string, error) {
	if v, ok := m[s]; ok {
		switch v.(type) {
		case string:
			return v.(string), nil
		default:
			return "", ErrInvalidFieldValue{Name: s}
		}
	}

	return "", nil
}

func extractStringList(m map[string]interface{}, s string) ([]string, error) {
	if v, ok := m[s]; ok {
		switch v.(type) {
		case string:
			return []string{v.(string)}, nil
		case []interface{}:
			l := v.([]interface{})
			r := make([]string, len(l))
			for i, x := range l {
				switch x.(type) {
				case string:
					r[i] = x.(string)
				default:
					return nil, ErrInvalidFieldValue{Name: s}
				}
			}
			return r, nil
		default:
			return nil, ErrInvalidFieldValue{Name: s}
		}
	}

	return nil, nil
}

func extractFormat(m map[string]interface{}, s string) (Format, error) {
	v, err := extractString(m, s)
	if err != nil {
		return "", err
	}
	return Format(v), nil
}

func extractJSPointer(m map[string]interface{}, s string) (string, error) {
	v, err := extractString(m, s)
	if err != nil {
		return "", err
	}

	return v, nil
}

func extractInterface(m map[string]interface{}, s string) (interface{}, error) {
	if v, ok := m[s]; ok {
		return v, nil
	}
	return nil, nil
}

func extractInterfaceList(m map[string]interface{}, s string) ([]interface{}, error) {
	if v, ok := m[s]; ok {
		switch v.(type) {
		case []interface{}:
			return v.([]interface{}), nil
		default:
			return nil, ErrInvalidFieldValue{Name: s}
		}
	}

	return nil, nil
}

func extractSchema(m map[string]interface{}) (*Schema, error) {
	s := New()
	if err := s.extract(m); err != nil {
		return nil, err
	}
	return s, nil
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

func (s *Schema) UnmarshalJSON(data []byte) error {
	m := map[string]interface{}{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	return s.extract(m)
}

func (s *Schema) extract(m map[string]interface{}) error {
	var err error

	if s.id, err = extractString(m, "id"); err != nil {
		return err
	}

	if s.Title, err = extractString(m, "title"); err != nil {
		return err
	}

	if s.Description, err = extractString(m, "description"); err != nil {
		return err
	}

	if s.Required, err = extractStringList(m, "required"); err != nil {
		return err
	}

	if s.SchemaRef, err = extractJSPointer(m, "$schema"); err != nil {
		return err
	}

	if s.Reference, err = extractJSPointer(m, "$ref"); err != nil {
		return err
	}

	if s.Format, err = extractFormat(m, "format"); err != nil {
		return err
	}

	if s.Enum, err = extractInterfaceList(m, "enum"); err != nil {
		return err
	}

	if s.Default, err = extractInterface(m, "default"); err != nil {
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

	if s.Items, err = extractSchemaList(m, "items"); err != nil {
		return err
	}

	if extractInt(&s.minItems, m, "minItems"); err != nil {
		return err
	}

	if extractInt(&s.maxItems, m, "maxItems"); err != nil {
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

	if s.properties, err = extractSchemaMap(m, "properties"); err != nil {
		return err
	}

	if s.AllOf, err = extractSchemaList(m, "allOf"); err != nil {
		return err
	}

	s.applyParentSchema()

	return nil
}

func placeString(m map[string]interface{}, name, s string) {
	if s != "" {
		m[name] = s
	}
}

func placeList(m map[string]interface{}, name string, l []interface{}) {
	if len(l) > 0 {
		m[name] = l
	}
}
func placeSchemaList(m map[string]interface{}, name string, l []*Schema) {
	if len(l) > 0 {
		m[name] = l
	}
}

func placeSchemaMap(m map[string]interface{}, name string, l map[string]*Schema) {
	if len(l) > 0 {
		defs := make(map[string]*Schema)
		m[name] = defs

		for k, v := range l {
			defs[k] = v
		}
	}
}

func placeStringList(m map[string]interface{}, name string, l []string) {
	if len(l) > 0 {
		m[name] = l
	}
}

func placeBool(m map[string]interface{}, name string, value bool, def bool) {
	if value == def { // no need to record default values
		return
	}
	m[name] = value
}

func placeNumber(m map[string]interface{}, name string, n Number) {
	if !n.Initialized {
		return
	}
	m[name] = n.Val
}

func placeInteger(m map[string]interface{}, name string, n integer) {
	if !n.Initialized {
		return
	}
	m[name] = n.Val
}

func (s Schema) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})

	placeString(m, "id", s.id)
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

	if s.AllowAdditionalItems {
		m["additionalItems"] = true
	}

	placeInteger(m, "maxItems", s.maxItems)
	placeInteger(m, "minItems", s.minItems)
	placeInteger(m, "maxProperties", s.MaxProperties)
	placeInteger(m, "minProperties", s.MinProperties)
	placeBool(m, "uniqueItems", s.UniqueItems, false)
	placeSchemaMap(m, "definitions", s.Definitions)

	switch len(s.Items) {
	case 0: // do nothing
	case 1:
		m["items"] = s.Items[0]
	case 2:
		m["items"] = s.Items
	}

	placeSchemaMap(m, "properties", s.properties)
	placeSchemaList(m, "allOf", s.AllOf)

	if s.Default != nil {
		m["default"] = s.Default
	}

	placeString(m, "format", string(s.Format))
	placeNumber(m, "minimum", s.Minimum)
	placeBool(m, "exclusiveminimum", s.ExclusiveMinimum, false)
	placeNumber(m, "Maximum", s.Maximum)
	placeBool(m, "exclusivemaximum", s.ExclusiveMaximum, false)

	return json.Marshal(m)
}
