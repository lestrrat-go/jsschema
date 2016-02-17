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

func (e ErrPatternValidationFailed) Error() string {
	return fmt.Sprintf("pattern did not match: '%s' does not match '%s'", e.Str, e.Pattern)
}
