package schema

import (
	"errors"
	"regexp"
	"sync"

	"github.com/lestrrat/go-jsref"
)

const (
	SchemaURL      = `http://json-schema.org/draft-04/schema`
	HyperSchemaURL = `http://json-schema.org/draft-03/hyper-schema`
	MIMEType       = "application/schema+json"
)

var ErrExpectedArrayOfString = errors.New("invalid value: expected array of string")
var ErrInvalidStringArray = ErrExpectedArrayOfString

type PrimitiveType int
type PrimitiveTypes []PrimitiveType
type Format string

const (
	FormatDateTime Format = "date-time"
	FormatEmail    Format = "email"
	FormatHostname Format = "hostname"
	FormatIPv4     Format = "ipv4"
	FormatIPv6     Format = "ipv6"
	FormatURI      Format = "uri"
)

type Number struct {
	Val         float64
	Initialized bool
}

type Integer struct {
	Val         int
	Initialized bool
}

type Bool struct {
	Val         bool
	Default     bool
	Initialized bool
}

const (
	UnspecifiedType PrimitiveType = iota
	NullType
	IntegerType
	StringType
	ObjectType
	ArrayType
	BooleanType
	NumberType
)

type SchemaList []*Schema
type Schema struct {
	parent          *Schema
	resolveLock     sync.Mutex
	resolvedSchemas map[string]interface{}
	resolver        *jsref.Resolver
	ID              string             `json:"id,omitempty"`
	Title           string             `json:"title,omitempty"`
	Description     string             `json:"description,omitempty"`
	Default         interface{}        `json:"default,omitempty"`
	Type            PrimitiveTypes     `json:"type,omitempty"`
	SchemaRef       string             `json:"$schema,omitempty"`
	Definitions     map[string]*Schema `json:"definitions,omitempty"`
	Reference       string             `json:"$ref,omitempty"`
	Format          Format             `json:"format,omitempty"`

	// NumericValidations
	MultipleOf       Number `json:"multipleOf,omitempty"`
	Minimum          Number `json:"minimum,omitempty"`
	Maximum          Number `json:"maximum,omitempty"`
	ExclusiveMinimum Bool   `json:"exclusiveMinimum,omitempty"`
	ExclusiveMaximum Bool   `json:"exclusiveMaximum,omitempty"`

	// StringValidation
	MaxLength Integer        `json:"maxLength,omitempty"`
	MinLength Integer        `json:"minLength,omitempty"`
	Pattern   *regexp.Regexp `json:"pattern,omitempty"`

	// ArrayValidations
	AdditionalItems *AdditionalItems
	Items           *ItemSpec
	MinItems        Integer
	MaxItems        Integer
	UniqueItems     Bool

	// ObjectValidations
	MaxProperties        Integer                    `json:"maxProperties,omitempty"`
	MinProperties        Integer                    `json:"minProperties,omitempty"`
	Required             []string                   `json:"required,omitempty"`
	Dependencies         DependencyMap              `json:"dependencies,omitempty"`
	Properties           map[string]*Schema         `json:"properties,omitempty"`
	AdditionalProperties *AdditionalProperties      `json:"additionalProperties,omitempty"`
	PatternProperties    map[*regexp.Regexp]*Schema `json:"patternProperties,omitempty"`

	Enum   []interface{}          `json:"enum,omitempty"`
	AllOf  SchemaList             `json:"allOf,omitempty"`
	AnyOf  SchemaList             `json:"anyOf,omitempty"`
	OneOf  SchemaList             `json:"oneOf,omitempty"`
	Not    *Schema                `json:"not,omitempty"`
	Extras map[string]interface{} `json:"-"`
}

type AdditionalItems struct {
	*Schema
}

type AdditionalProperties struct {
	*Schema
}

// DependencyMap contains the dependencies defined within this schema.
// for a given dependency name, you can have either a schema or a
// list of property names
type DependencyMap struct {
	Names   map[string][]string
	Schemas map[string]*Schema
}

type ItemSpec struct {
	TupleMode bool // If this is true, the positions mean something. if false, len(Schemas) should be 1, and we should apply the same schema validation to all elements
	Schemas   SchemaList
}
