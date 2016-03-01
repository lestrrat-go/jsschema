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

var (
	ErrAdditionalProperties        = errors.New("additional properties are not allowed")
	ErrAnyOfValidationFailed       = errors.New("'anyOf' validation failed")
	ErrArrayItemValidationFailed   = errors.New("'array' validation failed")
	ErrInvalidStringArray          = errors.New("invalid value: expected array of string")
	ErrDependencyItemType          = errors.New("'dependency' elements must be a array of strings or a schema")
	ErrOneOfValidationFailed       = errors.New("'oneOf' validation failed")
	ErrIntegerValidationFailed     = errors.New("'integer' validation failed")
	ErrInvalidEnum                 = errors.New("invalid enum type")
	ErrInvalidFormat               = errors.New("invalid format")
	ErrInvalidHostname             = errors.New("invalid hostname")
	ErrInvalidIPv4                 = errors.New("invalid IPv4 address")
	ErrInvalidIPv6                 = errors.New("invalid IPv6 address")
	ErrInvalidSchemaList           = errors.New("invalid schema list")
	ErrInvalidType                 = errors.New("invalid type")
	ErrMissingDependency           = errors.New("missing dependency")
	ErrMultipleOfValidationFailed  = errors.New("'multipleOf' validation failed")
	ErrNotValidationFailed         = errors.New("'not' validation failed")
	ErrNumberValidationFailed      = errors.New("'number' validation failed")
	ErrPropNotFound                = errors.New("property not found")
	ErrUniqueItemsValidationFailed = errors.New("'uniqueItems' validation failed")
	ErrSchemaNotFound              = errors.New("schema not found")
)

type PrimitiveType int
type PrimitiveTypes []PrimitiveType
type Format string
type ErrRequiredField struct {
	Name string
}
type ErrExtract struct {
	Field string
	Err   error
}

type ErrInvalidFieldValue struct {
	Name string
	Kind string
}
type ErrInvalidReference struct {
	Reference string
	Message   string
}
type ErrMinimumValidationFailed struct {
	Num       float64
	Min       float64
	Exclusive bool
}
type ErrMaximumValidationFailed struct {
	Num       float64
	Max       float64
	Exclusive bool
}
type ErrMinLengthValidationFailed struct {
	Len       int
	MinLength int
}
type ErrMaxLengthValidationFailed struct {
	Len       int
	MaxLength int
}
type ErrMinItemsValidationFailed struct {
	Len      int
	MinItems int
}
type ErrMaxItemsValidationFailed struct {
	Len      int
	MaxItems int
}
type ErrMinPropertiesValidationFailed struct {
	Num int
	Min int
}
type ErrMaxPropertiesValidationFailed struct {
	Num int
	Max int
}
type ErrPatternValidationFailed struct {
	Str     string
	Pattern *regexp.Regexp
}

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
