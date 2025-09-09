package searchx

import "github.com/cockroachdb/errors"

// Operator represents comparison operators.
type Operator string

const (
	// OpEq represents equality operator.
	OpEq Operator = "eq"
	// OpNe represents not-equal operator.
	OpNe Operator = "ne"
	// OpGt represents greater-than operator.
	OpGt Operator = "gt"
	// OpGte represents greater-than-or-equal operator.
	OpGte Operator = "gte"
	// OpLt represents less-than operator.
	OpLt Operator = "lt"
	// OpLte represents less-than-or-equal operator.
	OpLte Operator = "lte"
	// OpExists represents field existence check.
	OpExists Operator = "exists"
)

// ErrorCode represents specific error codes for search operations.
type ErrorCode int

const (
	// ErrCodeEmptyQuery is returned when an empty query is provided.
	ErrCodeEmptyQuery ErrorCode = iota + 1000

	// ErrCodeInvalidOption is returned when an invalid option is provided.
	ErrCodeInvalidOption

	// ErrCodeInvalidExpression is returned when an invalid expression is provided.
	ErrCodeInvalidExpression

	// ErrCodeTimeout is returned when a search operation times out.
	ErrCodeTimeout

	// ErrCodeCanceled is returned when a search operation is canceled.
	ErrCodeCanceled

	// ErrCodeNotImplemented is returned when a feature is not implemented.
	ErrCodeNotImplemented

	// ErrCodeBackendUnavailable is returned when the search backend is unavailable.
	ErrCodeBackendUnavailable
)

// String returns the human-readable string representation of the error code.
// This implements the fmt.Stringer interface.
func (e ErrorCode) String() string {
	switch e {
	case ErrCodeEmptyQuery:
		return "empty query"
	case ErrCodeInvalidOption:
		return "invalid option"
	case ErrCodeInvalidExpression:
		return "invalid expression"
	case ErrCodeTimeout:
		return "operation timed out"
	case ErrCodeCanceled:
		return "operation canceled"
	case ErrCodeNotImplemented:
		return "not implemented"
	case ErrCodeBackendUnavailable:
		return "backend unavailable"
	default:
		return "unknown error"
	}
}

// newErrorWithCode creates a new error with a code and message.
func newErrorWithCode(code ErrorCode, msg string) error {
	err := errors.New(msg)
	return errors.WithSecondaryError(err, errors.Newf("code: %d", int(code)))
}

// Common errors that can be returned by search operations.
var (
	// ErrEmptyQuery is returned when an empty query is provided.
	ErrEmptyQuery = newErrorWithCode(ErrCodeEmptyQuery, "searchx: empty query")

	// ErrInvalidOption is returned when an invalid option is provided.
	ErrInvalidOption = newErrorWithCode(ErrCodeInvalidOption, "searchx: invalid option")

	// ErrInvalidExpression is returned when an invalid expression is provided.
	ErrInvalidExpression = newErrorWithCode(ErrCodeInvalidExpression, "searchx: invalid expression")

	// ErrTimeout is returned when a search operation times out.
	ErrTimeout = newErrorWithCode(ErrCodeTimeout, "searchx: operation timed out")

	// ErrCanceled is returned when a search operation is canceled.
	ErrCanceled = newErrorWithCode(ErrCodeCanceled, "searchx: operation canceled")

	// ErrNotImplemented is returned when a feature is not implemented.
	ErrNotImplemented = newErrorWithCode(ErrCodeNotImplemented, "searchx: not implemented")

	// ErrBackendUnavailable is returned when the search backend is unavailable.
	ErrBackendUnavailable = newErrorWithCode(ErrCodeBackendUnavailable, "searchx: backend unavailable")
)
