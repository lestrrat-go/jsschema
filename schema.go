package schema

import (
	"encoding/json"
	"io"
	"net/url"
	"os"
	"reflect"
	"strings"

	"github.com/lestrrat/go-jsref"
	"github.com/lestrrat/go-jsref/provider"
	"github.com/lestrrat/go-pdebug"
)

// This is used to check against result of reflect.MapIndex
var zeroval = reflect.Value{}
var _schema Schema
var _hyperSchema Schema

func init() {
	buildJSSchema()
	buildHyperSchema()
}

func New() *Schema {
	s := Schema{}
	s.initialize()
	return &s
}

func (s *Schema) initialize() {
	resolver := jsref.New()

	mp := provider.NewMap()
	mp.Set(SchemaURL, &_schema)
	mp.Set(HyperSchemaURL, &_hyperSchema)
	resolver.AddProvider(mp)

	s.resolvedSchemas = make(map[string]interface{})
	s.resolver = resolver
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
	if err := s.decode(in); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Schema) decode(in io.Reader) error {
	dec := json.NewDecoder(in)
	if err := dec.Decode(s); err != nil {
		return err
	}
	s.applyParentSchema()
	return nil
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

// Resolve returns the schema after it has been resolved.
// If s.Reference is the empty string, the current schema is returned.
func (s *Schema) Resolve() (ref *Schema, err error) {
	if s.Reference == "" {
		return s, nil
	}

	if pdebug.Enabled {
		g := pdebug.IPrintf("START Schema.Resolve (%s)", s.Reference)
		defer func() {
			if err != nil {
				g.IRelease("END Schema.Resolve (%s): %s", s.Reference, err)
			} else {
				g.IRelease("END Schema.Resolve (%s)", s.Reference)
			}
		}()
	}

	var thing interface{}
	var ok bool
	s.resolveLock.Lock()
	thing, ok = s.resolvedSchemas[s.Reference]
	s.resolveLock.Unlock()

	if ok {
		ref, ok = thing.(*Schema)
		if ok {
			if pdebug.Enabled {
				pdebug.Printf("Cache HIT on '%s'", s.Reference)
			}
		} else {
			if pdebug.Enabled {
				pdebug.Printf("Negative Cache HIT on '%s'", s.Reference)
			}
			return nil, thing.(error)
		}
	} else {
		if pdebug.Enabled {
			pdebug.Printf("Cache MISS on '%s'", s.Reference)
		}
		var err error
		thing, err := s.resolver.Resolve(s.Root(), s.Reference)
		if err != nil {
			err = ErrInvalidReference{Reference: s.Reference, Message: err.Error()}
			s.resolveLock.Lock()
			s.resolvedSchemas[s.Reference] = err
			s.resolveLock.Unlock()
			return nil, err
		}

		ref, ok = thing.(*Schema)
		if !ok {
			err = ErrInvalidReference{Reference: s.Reference, Message: "returned element is not a Schema"}
			s.resolveLock.Lock()
			s.resolvedSchemas[s.Reference] = err
			s.resolveLock.Unlock()
			return nil, err
		}
		s.resolveLock.Lock()
		s.resolvedSchemas[s.Reference] = ref
		s.resolveLock.Unlock()
	}

	return ref, nil
}

func (s Schema) IsPropRequired(pname string) bool {
	for _, name := range s.Required {
		if name == pname {
			return true
		}
	}
	return false
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
	if err := _schema.decode(strings.NewReader(src)); err != nil {
		// We regret to inform you that if we can't parse this
		// schema, then we have a real real real problem, so we're
		// going to panic
		panic("failed to parse main JSON Schema schema: " + err.Error())
	}
}

func buildHyperSchema() {
	const src = `{
  "$schema": "http://json-schema.org/draft-04/hyper-schema#",
  "id": "http://json-schema.org/draft-04/hyper-schema#",
  "title": "JSON Hyper-Schema",
  "allOf": [
    {
      "$ref": "http://json-schema.org/draft-04/schema#"
    }
  ],
  "properties": {
    "additionalItems": {
      "anyOf": [
        {
          "type": "boolean"
        },
        {
          "$ref": "#"
        }
      ]
    },
    "additionalProperties": {
      "anyOf": [
        {
          "type": "boolean"
        },
        {
          "$ref": "#"
        }
      ]
    },
    "dependencies": {
      "additionalProperties": {
        "anyOf": [
          {
            "$ref": "#"
          },
          {
            "type": "array"
          }
        ]
      }
    },
    "items": {
      "anyOf": [
        {
          "$ref": "#"
        },
        {
          "$ref": "#/definitions/schemaArray"
        }
      ]
    },
    "definitions": {
      "additionalProperties": {
        "$ref": "#"
      }
    },
    "patternProperties": {
      "additionalProperties": {
        "$ref": "#"
      }
    },
    "properties": {
      "additionalProperties": {
        "$ref": "#"
      }
    },
    "allOf": {
      "$ref": "#/definitions/schemaArray"
    },
    "anyOf": {
      "$ref": "#/definitions/schemaArray"
    },
    "oneOf": {
      "$ref": "#/definitions/schemaArray"
    },
    "not": {
      "$ref": "#"
    },
    "links": {
      "type": "array",
      "items": {
        "$ref": "#/definitions/linkDescription"
      }
    },
    "fragmentResolution": {
      "type": "string"
    },
    "media": {
      "type": "object",
      "properties": {
        "type": {
          "description": "A media type, as described in RFC 2046",
          "type": "string"
        },
        "binaryEncoding": {
          "description": "A content encoding scheme, as described in RFC 2045",
          "type": "string"
        }
      }
    },
    "pathStart": {
      "description": "Instances' URIs must start with this value for this schema to apply to them",
      "type": "string",
      "format": "uri"
    }
  },
  "definitions": {
    "schemaArray": {
      "type": "array",
      "items": {
        "$ref": "#"
      }
    },
    "linkDescription": {
      "title": "Link Description Object",
      "type": "object",
      "required": [
        "href",
        "rel"
      ],
      "properties": {
        "href": {
          "description": "a URI template, as defined by RFC 6570, with the addition of the $, ( and ) characters for pre-processing",
          "type": "string"
        },
        "rel": {
          "description": "relation to the target resource of the link",
          "type": "string"
        },
        "title": {
          "description": "a title for the link",
          "type": "string"
        },
        "targetSchema": {
          "description": "JSON Schema describing the link target",
          "$ref": "#"
        },
        "mediaType": {
          "description": "media type (as defined by RFC 2046) describing the link target",
          "type": "string"
        },
        "method": {
          "description": "method for requesting the target of the link (e.g. for HTTP this might be \"GET\" or \"DELETE\")",
          "type": "string"
        },
        "encType": {
          "description": "The media type in which to submit data along with the request",
          "type": "string",
          "default": "application/json"
        },
        "schema": {
          "description": "Schema describing the data to submit along with the request",
          "$ref": "#"
        }
      }
    }
  },
  "links": [
    {
      "rel": "self",
      "href": "{+id}"
    },
    {
      "rel": "full",
      "href": "{+($ref)}"
    }
  ]
}`
	if err := _hyperSchema.decode(strings.NewReader(src)); err != nil {
		// We regret to inform you that if we can't parse this
		// schema, then we have a real real real problem, so we're
		// going to panic
		panic("failed to parse Hyper JSON Schema schema: " + err.Error())
	}
}
