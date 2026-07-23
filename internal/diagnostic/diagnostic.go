// Package diagnostic defines structured source diagnostics shared by compiler phases.
package diagnostic

import (
	"cmp"
	"fmt"
	"slices"
)

// Position identifies a byte and its human-readable location in a source file.
type Position struct {
	Offset int
	Line   int
	Column int
}

// Span identifies a half-open range in a source file.
type Span struct {
	Path  string
	Start Position
	End   Position
}

// Severity describes the effect of a diagnostic.
type Severity uint8

const (
	// SeverityError marks input that cannot be compiled.
	SeverityError Severity = iota + 1
	// SeverityWarning marks valid input that may be unintended.
	SeverityWarning
)

// Diagnostic is a machine-readable compiler finding.
type Diagnostic struct {
	Span     Span
	Severity Severity
	Code     string
	Message  string
}

// Error constructs an error diagnostic at span.
func Error(span Span, code, message string) Diagnostic {
	return Diagnostic{Span: span, Severity: SeverityError, Code: code, Message: message}
}

// Sort orders diagnostics by source position, severity, code, and message.
func Sort(ds []Diagnostic) {
	slices.SortFunc(ds, func(a, b Diagnostic) int {
		if n := cmp.Compare(a.Span.Path, b.Span.Path); n != 0 {
			return n
		}
		if n := cmp.Compare(a.Span.Start.Offset, b.Span.Start.Offset); n != 0 {
			return n
		}
		if n := cmp.Compare(a.Severity, b.Severity); n != 0 {
			return n
		}
		if n := cmp.Compare(a.Code, b.Code); n != 0 {
			return n
		}
		return cmp.Compare(a.Message, b.Message)
	})
}

// String renders a stable single-line human-readable diagnostic.
func (d Diagnostic) String() string {
	return fmt.Sprintf("%s:%d:%d %s %s", d.Span.Path, d.Span.Start.Line, d.Span.Start.Column, d.Code, d.Message)
}
