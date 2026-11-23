package core

import (
	"errors"
	"fmt"
	"slices"
)

// ErrInvalidResource indicates a Resource failed structural validation.
var ErrInvalidResource = errors.New("invalid resource")

// DriverKind identifies one of Mirolu's five resource drivers.
type DriverKind string

const (
	DriverFile         DriverKind = "file"
	DriverPackage      DriverKind = "package"
	DriverService      DriverKind = "service"
	DriverSecretTarget DriverKind = "secret_target"
	DriverCommand      DriverKind = "command"
)

// Valid reports whether k is one of the five supported DriverKind values.
func (k DriverKind) Valid() bool {
	switch k {
	case DriverFile, DriverPackage, DriverService, DriverSecretTarget, DriverCommand:
		return true
	}
	return false
}

// Resource is an atomic, idempotent unit of desired system state.
type Resource struct {
	ID       string     `json:"id"`
	Module   string     `json:"module"`
	Kind     DriverKind `json:"kind"`
	Spec     any        `json:"-"`
	Requires []string   `json:"requires,omitempty"`
	Notifies []string   `json:"notifies,omitempty"`
}

// Validate checks that r has required fields, a valid driver kind, and non-empty edges.
func (r Resource) Validate() error {
	switch {
	case r.ID == "":
		return fmt.Errorf("%w: empty id", ErrInvalidResource)
	case r.Module == "":
		return fmt.Errorf("%w: %q has empty module", ErrInvalidResource, r.ID)
	case !r.Kind.Valid():
		return fmt.Errorf("%w: %q has unknown kind %q", ErrInvalidResource, r.ID, r.Kind)
	case slices.Contains(r.Requires, ""):
		return fmt.Errorf("%w: %q has empty requires entry", ErrInvalidResource, r.ID)
	case slices.Contains(r.Notifies, ""):
		return fmt.Errorf("%w: %q has empty notifies entry", ErrInvalidResource, r.ID)
	default:
		return nil
	}
}
