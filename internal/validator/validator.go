package validator

import "regexp"

var (
	EmailRX  = regexp.MustCompile("^[a-zA-Z0-9._-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+$")
	UserIDRX = regexp.MustCompile("^ID-[0-9]{1,32}$")
)

type Validator struct {
	Errors map[string]string
}

// New creates and returns a new Validator
func New() *Validator {
	return &Validator{
		Errors: make(map[string]string),
	}
}

// Valid returns True if there is no error, otherwise false.
func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

// AddError adds an error message to the map, if it's not already there.
func (v *Validator) AddError(key string, message string) {
	if _, exists := v.Errors[key]; !exists {
		v.Errors[key] = message
	}
}

// Check checks if the ok is true, if not, it adds an error message to the map.
func (v *Validator) Check(ok bool, key string, message string) {
	if !ok {
		v.AddError(key, message)
	}
}

// IsPermittedValue returns true if a specific value is in a list of permitted values.
func IsPermittedValue[T comparable](value T, permittedValues []T) bool {
	for _, v := range permittedValues {
		if v == value {
			return true
		}
	}
	return false
}

// Matches return true if a string satisfies a specific regular expression pattern.
func Matches(value string, regex *regexp.Regexp) bool {
	return regex.MatchString(value)
}

// Unique returns true if all values in a slice are unique.
func Unique[T comparable](values []T) bool {
	uniqueValues := make(map[T]bool)
	for _, v := range values {
		uniqueValues[v] = true
	}
	return len(values) == len(uniqueValues)
}
