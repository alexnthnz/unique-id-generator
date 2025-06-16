package generator

import (
	"fmt"
	"time"
)

// ErrorType represents different types of errors that can occur
type ErrorType int

const (
	ErrorTypeInvalidNodeID ErrorType = iota
	ErrorTypeClockBackward
	ErrorTypeSequenceExhausted
	ErrorTypeTimestampExhausted
	ErrorTypeGenerationFailed
	ErrorTypeInvalidConfiguration
)

// String returns a string representation of the error type
func (et ErrorType) String() string {
	switch et {
	case ErrorTypeInvalidNodeID:
		return "InvalidNodeID"
	case ErrorTypeClockBackward:
		return "ClockBackward"
	case ErrorTypeSequenceExhausted:
		return "SequenceExhausted"
	case ErrorTypeTimestampExhausted:
		return "TimestampExhausted"
	case ErrorTypeGenerationFailed:
		return "GenerationFailed"
	case ErrorTypeInvalidConfiguration:
		return "InvalidConfiguration"
	default:
		return "Unknown"
	}
}

// IDGeneratorError represents a structured error from the ID generator
type IDGeneratorError struct {
	Type      ErrorType              `json:"type"`
	Message   string                 `json:"message"`
	Cause     error                  `json:"cause,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	NodeID    uint16                 `json:"node_id,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// Error implements the error interface
func (e *IDGeneratorError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type.String(), e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type.String(), e.Message)
}

// Unwrap returns the underlying cause error
func (e *IDGeneratorError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches the target error type
func (e *IDGeneratorError) Is(target error) bool {
	if targetErr, ok := target.(*IDGeneratorError); ok {
		return e.Type == targetErr.Type
	}
	return false
}

// NewIDGeneratorError creates a new structured error
func NewIDGeneratorError(errType ErrorType, message string) *IDGeneratorError {
	return &IDGeneratorError{
		Type:      errType,
		Message:   message,
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
	}
}

// NewIDGeneratorErrorWithCause creates a new structured error with a cause
func NewIDGeneratorErrorWithCause(errType ErrorType, message string, cause error) *IDGeneratorError {
	return &IDGeneratorError{
		Type:      errType,
		Message:   message,
		Cause:     cause,
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
	}
}

// WithNodeID adds node ID context to the error
func (e *IDGeneratorError) WithNodeID(nodeID uint16) *IDGeneratorError {
	e.NodeID = nodeID
	return e
}

// WithContext adds additional context to the error
func (e *IDGeneratorError) WithContext(key string, value interface{}) *IDGeneratorError {
	e.Context[key] = value
	return e
}

// Predefined errors for common cases
var (
	ErrInvalidNodeID      = NewIDGeneratorError(ErrorTypeInvalidNodeID, "node ID must be between 0 and 1023")
	ErrClockMovedBackward = NewIDGeneratorError(ErrorTypeClockBackward, "clock moved backward")
	ErrSequenceExhausted  = NewIDGeneratorError(ErrorTypeSequenceExhausted, "sequence exhausted for current millisecond")
	ErrTimestampExhausted = NewIDGeneratorError(ErrorTypeTimestampExhausted, "timestamp exhausted - epoch overflow")
)

// IsClockBackwardError checks if an error is a clock backward error
func IsClockBackwardError(err error) bool {
	var idErr *IDGeneratorError
	return err != nil &&
		(err == ErrClockMovedBackward ||
			(AsIDGeneratorError(err, &idErr) && idErr.Type == ErrorTypeClockBackward))
}

// IsSequenceExhaustedError checks if an error is a sequence exhausted error
func IsSequenceExhaustedError(err error) bool {
	var idErr *IDGeneratorError
	return err != nil &&
		(err == ErrSequenceExhausted ||
			(AsIDGeneratorError(err, &idErr) && idErr.Type == ErrorTypeSequenceExhausted))
}

// AsIDGeneratorError attempts to cast an error to IDGeneratorError
func AsIDGeneratorError(err error, target **IDGeneratorError) bool {
	if err == nil {
		return false
	}
	if idErr, ok := err.(*IDGeneratorError); ok {
		*target = idErr
		return true
	}
	return false
}
