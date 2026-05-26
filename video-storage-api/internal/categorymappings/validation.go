package categorymappings

import "fmt"

type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	if e == nil {
		return "invalid category"
	}
	return fmt.Sprintf("invalid category: %s: %s", e.Field, e.Reason)
}

func invalidCategory(field, reason string) error {
	return &ValidationError{Field: field, Reason: reason}
}
