package errors

import (
	"fmt"

	"github.com/jafarshop/b2bapi/internal/domain"
)

// ErrNotFound is returned when a resource is not found
type ErrNotFound struct {
	Resource string
	ID       string
}

func (e *ErrNotFound) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
}

// ErrUnauthorized is returned when authentication fails
type ErrUnauthorized struct {
	Message string
}

func (e *ErrUnauthorized) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "unauthorized"
}

// ErrConflict is returned when there's a conflict (e.g., idempotency)
type ErrConflict struct {
	Message string
}

func (e *ErrConflict) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "conflict"
}

// ErrValidation is returned when validation fails
type ErrValidation struct {
	Message string
	Fields  map[string]string
}

func (e *ErrValidation) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "validation failed"
}

// ErrInvalidStateTransition is returned when an invalid state transition is attempted
type ErrInvalidStateTransition struct {
	From domain.OrderStatus
	To   domain.OrderStatus
}

func (e *ErrInvalidStateTransition) Error() string {
	return fmt.Sprintf("invalid state transition from %s to %s", e.From, e.To)
}
