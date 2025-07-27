package errors

import (
	"errors"
	"fmt"
)

// Error types
var (
	ErrNotFound    = errors.New("resource not found")
	ErrValidation  = errors.New("validation error")
	ErrConflict    = errors.New("resource conflict")
	ErrInternal    = errors.New("internal error")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden   = errors.New("forbidden")
)

// AppError represents an application error with additional context
type AppError struct {
	Type    error
	Message string
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type.Error(), e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type.Error(), e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(message string, cause error) *AppError {
	return &AppError{
		Type:    ErrNotFound,
		Message: message,
		Cause:   cause,
	}
}

// NewValidationError creates a new validation error
func NewValidationError(message string, cause error) *AppError {
	return &AppError{
		Type:    ErrValidation,
		Message: message,
		Cause:   cause,
	}
}

// NewConflictError creates a new conflict error
func NewConflictError(message string, cause error) *AppError {
	return &AppError{
		Type:    ErrConflict,
		Message: message,
		Cause:   cause,
	}
}

// NewInternalError creates a new internal error
func NewInternalError(message string, cause error) *AppError {
	return &AppError{
		Type:    ErrInternal,
		Message: message,
		Cause:   cause,
	}
}

// NewUnauthorizedError creates a new unauthorized error
func NewUnauthorizedError(message string, cause error) *AppError {
	return &AppError{
		Type:    ErrUnauthorized,
		Message: message,
		Cause:   cause,
	}
}

// NewForbiddenError creates a new forbidden error
func NewForbiddenError(message string, cause error) *AppError {
	return &AppError{
		Type:    ErrForbidden,
		Message: message,
		Cause:   cause,
	}
}

// Type checking functions
func IsNotFound(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return errors.Is(appErr.Type, ErrNotFound)
	}
	return false
}

func IsValidation(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return errors.Is(appErr.Type, ErrValidation)
	}
	return false
}

func IsConflict(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return errors.Is(appErr.Type, ErrConflict)
	}
	return false
}

func IsInternal(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return errors.Is(appErr.Type, ErrInternal)
	}
	return false
}

func IsUnauthorized(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return errors.Is(appErr.Type, ErrUnauthorized)
	}
	return false
}

func IsForbidden(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return errors.Is(appErr.Type, ErrForbidden)
	}
	return false
}