package schema

import (
	"bytes"
	"strconv"
)

func (e ErrExtract) Error() string {
	buf := bytes.Buffer{}
	buf.WriteString("failed to extract '")
	buf.WriteString(e.Field)
	buf.WriteString("' from JSON: ")
	buf.WriteString(e.Err.Error())
	return buf.String()
}

func (e ErrInvalidFieldValue) Error() string {
	buf := bytes.Buffer{}
	buf.WriteString("invalid value for field ")
	buf.WriteString(e.Name)
	buf.WriteString(" (")
	switch e.Value {
	case zeroval:
		buf.WriteString("invalid value")
	default:
		buf.WriteString(e.Value.Type().String())
	}
	buf.WriteByte(')')
	if msg := e.Message; msg != "" {
		buf.WriteString(": ")
		buf.WriteString(msg)
	}

	return buf.String()
}

func (e ErrInvalidReference) Error() string {
	buf := bytes.Buffer{}
	buf.WriteString("failed to resolve reference '")
	buf.WriteString(e.Reference)
	buf.WriteString("': ")
	buf.WriteString(e.Message)
	return buf.String()
}

func (e ErrRequiredField) Error() string {
	buf := bytes.Buffer{}
	buf.WriteString("required field '")
	buf.WriteString(e.Name)
	buf.WriteString("' not found")
	return buf.String()
}

func (e ErrMinLengthValidationFailed) Error() string {
	buf := bytes.Buffer{}
	buf.WriteString("required minimum length not met: ")
	buf.WriteString(strconv.Itoa(e.Len))
	buf.WriteString(" < ")
	buf.WriteString(strconv.Itoa(e.MinLength))
	return buf.String()
}

func (e ErrMaxLengthValidationFailed) Error() string {
	buf := bytes.Buffer{}
	buf.WriteString("required maximum length not met: ")
	buf.WriteString(strconv.Itoa(e.Len))
	buf.WriteString(" > ")
	buf.WriteString(strconv.Itoa(e.MaxLength))
	return buf.String()
}

func (e ErrMinItemsValidationFailed) Error() string {
	buf := bytes.Buffer{}
	buf.WriteString("required minimum item count not met: ")
	buf.WriteString(strconv.Itoa(e.Len))
	buf.WriteString(" < ")
	buf.WriteString(strconv.Itoa(e.MinItems))
	return buf.String()
}

func (e ErrMaxItemsValidationFailed) Error() string {
	buf := bytes.Buffer{}
	buf.WriteString("required maximum item count not met: ")
	buf.WriteString(strconv.Itoa(e.Len))
	buf.WriteString(" > ")
	buf.WriteString(strconv.Itoa(e.MaxItems))
	return buf.String()
}

func (e ErrMinPropertiesValidationFailed) Error() string {
	buf := bytes.Buffer{}
	buf.WriteString("number of properties fewer than minimum number: ")
	buf.WriteString(strconv.Itoa(e.Num))
	buf.WriteString(" < ")
	buf.WriteString(strconv.Itoa(e.Min))
	return buf.String()
}

func (e ErrMaxPropertiesValidationFailed) Error() string {
	buf := bytes.Buffer{}
	buf.WriteString("number of properties exceed maximum number: ")
	buf.WriteString(strconv.Itoa(e.Num))
	buf.WriteString(" > ")
	buf.WriteString(strconv.Itoa(e.Max))
	return buf.String()
}

func (e ErrPatternValidationFailed) Error() string {
	buf := bytes.Buffer{}
	buf.WriteString("pattern did not match: '")
	buf.WriteString(e.Str)
	buf.WriteString("' does not match '")
	buf.WriteString(e.Pattern.String())
	buf.WriteString("'")
	return buf.String()
}

func (e ErrMinimumValidationFailed) Error() string {
	buf := bytes.Buffer{}
	buf.WriteString("value is smaller than minimum: ")
	buf.WriteString(strconv.FormatFloat(e.Num, 'f', -1, 64))
	if e.Exclusive {
		buf.WriteString(" < ")
	} else {
		buf.WriteString(" <= ")
	}
	buf.WriteString(strconv.FormatFloat(e.Min, 'f', -1, 64))
	return buf.String()
}

func (e ErrMaximumValidationFailed) Error() string {
	buf := bytes.Buffer{}
	buf.WriteString("value exceeds maximum: ")
	buf.WriteString(strconv.FormatFloat(e.Num, 'f', -1, 64))
	if e.Exclusive {
		buf.WriteString(" > ")
	} else {
		buf.WriteString(" >= ")
	}
	buf.WriteString(strconv.FormatFloat(e.Max, 'f', -1, 64))
	return buf.String()
}
