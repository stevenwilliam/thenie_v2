// Package apierror is the single error model for the Thenie config service.
//
// Every error that reaches a client is an *Error carrying a stable code, a
// human-readable message and optional structured details. Driver errors,
// wrapped causes and stack traces are kept internally and never serialised.
package apierror

import (
	"errors"
	"fmt"
	"net/http"
)

// Code is a stable, machine-readable error identifier. Clients switch on these;
// they never parse messages.
type Code string

const (
	CodeValidation      Code = "VALIDATION_FAILED"
	CodeUnauthenticated Code = "UNAUTHENTICATED"
	CodeForbidden       Code = "FORBIDDEN"
	CodeNotFound        Code = "NOT_FOUND"
	CodeConflict        Code = "CONFLICT"
	CodeInternal        Code = "INTERNAL"

	// Content-specific. A rate table that breaks the front end's pricing
	// engine, or a cycle that overlaps a published one, is a conflict the
	// caller can act on -- not a generic 500.
	CodeRateInvariant  Code = "RATE_INVARIANT_VIOLATED"
	CodeCycleOverlap   Code = "CYCLE_OVERLAP"
	CodeCardKeyUnknown Code = "CARD_KEY_UNKNOWN"
	CodePublishBlocked Code = "PUBLISH_BLOCKED"
)

// Error is an application error with an HTTP status and a stable code.
type Error struct {
	Code    Code
	Message string
	Status  int
	Details map[string]any

	cause error // never serialised
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.cause }

// WithDetails attaches structured details. Details are client-visible, so they
// must never carry secrets or internal identifiers beyond resource ids.
func (e *Error) WithDetails(d map[string]any) *Error {
	c := *e
	c.Details = d
	return &c
}

// WithCause attaches an internal cause for logging. The cause is never rendered
// to a client.
func (e *Error) WithCause(err error) *Error {
	c := *e
	c.cause = err
	return &c
}

func New(status int, code Code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

func BadRequest(code Code, msg string) *Error { return New(http.StatusBadRequest, code, msg) }
func Unauthorized(msg string) *Error {
	return New(http.StatusUnauthorized, CodeUnauthenticated, msg)
}
func Forbidden(msg string) *Error           { return New(http.StatusForbidden, CodeForbidden, msg) }
func NotFound(msg string) *Error            { return New(http.StatusNotFound, CodeNotFound, msg) }
func Conflict(code Code, msg string) *Error { return New(http.StatusConflict, code, msg) }
func Unprocessable(code Code, msg string) *Error {
	return New(http.StatusUnprocessableEntity, code, msg)
}

// Internal wraps an unexpected error. The cause is logged; the client sees a
// generic message.
func Internal(err error) *Error {
	return &Error{
		Status:  http.StatusInternalServerError,
		Code:    CodeInternal,
		Message: "Something went wrong on our side.",
		cause:   err,
	}
}

// Validation reports a failed input validation with field details.
func Validation(msg string, fields map[string]any) *Error {
	return &Error{
		Status:  http.StatusBadRequest,
		Code:    CodeValidation,
		Message: msg,
		Details: fields,
	}
}

// From converts any error into an *Error, defaulting to Internal. It is the
// single funnel used by the HTTP error middleware.
func From(err error) *Error {
	if err == nil {
		return nil
	}
	var ae *Error
	if errors.As(err, &ae) {
		return ae
	}
	return Internal(err)
}

// Is reports whether err is an *Error with the given code.
func Is(err error, code Code) bool {
	var ae *Error
	return errors.As(err, &ae) && ae.Code == code
}
