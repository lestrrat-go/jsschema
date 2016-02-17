package schema

import "fmt"

func (e ErrInvalidFieldValue) Error() string {
	return fmt.Sprintf("invalid value for field %s", e.Name)
}

func (e ErrInvalidReference) Error() string {
	return fmt.Sprintf("failed to resolve reference '%s': %s", e.Reference, e.Message)
}

func (e ErrRequiredField) Error() string {
	return fmt.Sprintf("required field '%s' not found", e.Name)
}

func (e ErrMinLengthValidationFailed) Error() string {
	return fmt.Sprintf("required minimum length not met: %d < %d", e.Len, e.MinLength)
}

func (e ErrMaxLengthValidationFailed) Error() string {
	return fmt.Sprintf("required maximum length not met: %d > %d", e.Len, e.MaxLength)
}

func (e ErrMinItemsValidationFailed) Error() string {
	return fmt.Sprintf("required minimum item count not met: %d < %d", e.Len, e.MinItems)
}

func (e ErrMaxItemsValidationFailed) Error() string {
	return fmt.Sprintf("required maximum item count not met: %d > %d", e.Len, e.MaxItems)
}

func (e ErrMinPropertiesValidationFailed) Error() string {
	return fmt.Sprintf("number of properties fewer than minimum number: %d < %d", e.Num, e.Min)
}

func (e ErrMaxPropertiesValidationFailed) Error() string {
	return fmt.Sprintf("number of properties exceed maximum number: %d > %d", e.Num, e.Max)
}

func (e ErrPatternValidationFailed) Error() string {
	return fmt.Sprintf("pattern did not match: '%s' does not match '%s'", e.Str, e.Pattern)
}

func (e ErrMinimumValidationFailed) Error() string {
	sign := "<="
	if e.Exclusive {
		sign = "<"
	}
	return fmt.Sprintf("value exceeds minimum: %d %s %d", e.Num, sign, e.Min)
}

func (e ErrMaximumValidationFailed) Error() string {
	sign := ">="
	if e.Exclusive {
		sign = ">"
	}
	return fmt.Sprintf("value exceeds maximum: %d %s %d", e.Num, sign, e.Max)
}


