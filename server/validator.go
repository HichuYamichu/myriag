package server

import "github.com/go-playground/validator/v10"

// Validator used to validate bound structs
type Validator struct {
	validator *validator.Validate
}

// NewValidator creates new Validator
func NewValidator() *Validator {
	return &Validator{
		validator: validator.New(),
	}
}

// Validate satisfies Validator interface
func (v *Validator) Validate(i interface{}) error {
	return v.validator.Struct(i)
}
