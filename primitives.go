package schema

import (
	"encoding/json"
	"errors"
)

func (t *PrimitiveType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	x, err := primitiveFromString(string(data))
	if err != nil {
		return err
	}
	*t = x
	return nil
}

func primitiveFromString(s string) (t PrimitiveType, err error) {
	switch s {
	case "null":
		t = NullType
	case "integer":
		t = IntegerType
	case "string":
		t = StringType
	case "object":
		t = ObjectType
	case "array":
		t = ArrayType
	case "boolean":
		t = BooleanType
	case "number":
		t = NumberType
	default:
		err = errors.New("unknown primitive type: " + s)
	}
	return
}

func (t PrimitiveType) String() string {
	var v string
	switch t {
	case NullType:
		v = "null"
	case IntegerType:
		v = "integer"
	case StringType:
		v = "string"
	case ObjectType:
		v = "object"
	case ArrayType:
		v = "array"
	case BooleanType:
		v = "boolean"
	case NumberType:
		v = "number"
	default:
		v = "<invalid>"
	}
	return v
}

func (t PrimitiveType) MarshalJSON() ([]byte, error) {
	switch t {
	case NullType, IntegerType, StringType, ObjectType, ArrayType, BooleanType, NumberType:
		return json.Marshal(t.String())
	default:
		return nil, errors.New("unknown primitive type")
	}
}

func (ts *PrimitiveTypes) UnmarshalJSON(data []byte) error {
	if data[0] != '[' {
		var t PrimitiveType
		if err := json.Unmarshal(data, &t); err != nil {
			return err
		}

		*ts = PrimitiveTypes{t}
		return nil
	}

	var list []PrimitiveType
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}

	*ts = PrimitiveTypes(list)
	return nil
}

func (b Bool) Bool() bool {
	if b.Initialized {
		return b.Val
	}
	return b.Default
}
