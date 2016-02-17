package schema

import (
	"errors"
	"regexp"
)

const MIMEType = "application/schema+json"

var (
	ErrAdditionalProperties       = errors.New("additional properties are not allowed")
	ErrAnyOfValidationFailed      = errors.New("'anyOf' validation failed")
	ErrOneOfValidationFailed      = errors.New("'oneOf' validation failed")
	ErrIntegerValidationFailed    = errors.New("'integer' validation failed")
	ErrInvalidFormat              = errors.New("invalid format")
	ErrInvalidHostname            = errors.New("invalid hostname")
	ErrInvalidIPv4                = errors.New("invalid IPv4 address")
	ErrInvalidIPv6                = errors.New("invalid IPv6 address")
	ErrInvalidType                = errors.New("invalid type")
	ErrMultipleOfValidationFailed = errors.New("'multipleOf' validation failed")
	ErrNotValidationFailed        = errors.New("'not' validation failed")
	ErrNumberValidationFailed     = errors.New("'number' validation failed")
	ErrPropNotFound               = errors.New("property not found")
	ErrSchemaNotFound             = errors.New("schema not found")
)

type PrimitiveType int
type PrimitiveTypes []PrimitiveType
type Format string
type ErrRequiredField struct {
	Name string
}
type ErrInvalidFieldValue struct {
	Name string
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

type Schema struct {
	parent          *Schema
	cachedReference map[string]interface{}
	schemaByID      map[string]*Schema
	id              string             `json:"id,omitempty"`
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
	AllowAdditionalItems bool
	AdditionalItems      []*Schema
	Items                []*Schema
	minItems             Integer
	maxItems             Integer
	UniqueItems          Bool

	// ObjectValidations
	MaxProperties        Integer                    `json:"maxProperties,omitempty"`
	MinProperties        Integer                    `json:"minProperties,omitempty"`
	Required             []string                   `json:"required,omitempty"`
	properties           map[string]*Schema         `json:"properties,omitempty"`
	AdditionalProperties *AdditionalProperties      `json:"additionalProperties,omitempty"`
	PatternProperties    map[*regexp.Regexp]*Schema `json:"patternProperties,omitempty"`

	Enum  []interface{} `json:"enum,omitempty"`
	AllOf []*Schema     `json:"allOf,omitempty"`
	AnyOf []*Schema     `json:"anyOf,omitempty"`
	OneOf []*Schema     `json:"oneOf,omitempty"`
	Not   *Schema       `json:"not,omitempty"`
}

type AdditionalProperties struct {
	*Schema
}
