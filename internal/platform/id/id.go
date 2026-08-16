// Package id implements the application/id contract with time-ordered UUID v7
// identifiers. The concrete generator is selected here, never in application
// code.
package id

import (
	"github.com/google/uuid"
)

// Generator emits version-7 UUID strings.
// It is a stateless value receiver, so a zero value is ready to use.
type Generator struct{}

// New returns a version-7 UUID string (time-ordered, sorts by insertion).
// It panics if the underlying UUID generator is unavailable, which is not
// expected in practice.
func (Generator) New() string {
	v, err := uuid.NewV7()
	if err != nil {
		panic("id: uuid v7 unavailable: " + err.Error())
	}
	return v.String()
}
