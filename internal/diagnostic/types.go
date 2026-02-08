package diagnostic

import (
	"errors"
	"fmt"
	"strings"

	"caster-generator/internal/common"
)

// Diagnostics holds all diagnostic information from resolution.
type Diagnostics struct {
	Errors   []Diagnostic
	Warnings []Diagnostic
	Infos    []Diagnostic
}

// Diagnostic represents a single diagnostic message.
type Diagnostic struct {
	// Severity of the diagnostic.
	Severity DiagnosticSeverity
	// Code is a unique identifier for this type of diagnostic.
	Code string
	// Message is the human-readable description.
	Message string
	// TypePair identifies which type mapping this relates to (if any).
	TypePair string
	// FieldPath identifies which field this relates to (if any).
	FieldPath string
	// Suggestions are potential fixes or alternatives.
	Suggestions []string
}

// DiagnosticSeverity represents the severity level of a diagnostic.
type DiagnosticSeverity int

const (
	DiagnosticInfo DiagnosticSeverity = iota
	DiagnosticWarning
	DiagnosticError
)

// String returns a human-readable severity name.
func (s DiagnosticSeverity) String() string {
	switch s {
	case DiagnosticInfo:
		return "info"
	case DiagnosticWarning:
		return "warning"
	case DiagnosticError:
		return "error"
	default:
		return common.UnknownStr
	}
}

// AddError adds an error diagnostic.
func (d *Diagnostics) AddError(code, message, typePair, fieldPath string) {
	d.Errors = append(d.Errors, Diagnostic{
		Severity:  DiagnosticError,
		Code:      code,
		Message:   message,
		TypePair:  typePair,
		FieldPath: fieldPath,
	})
}

// AddWarning adds a warning diagnostic.
func (d *Diagnostics) AddWarning(code, message, typePair, fieldPath string) {
	d.Warnings = append(d.Warnings, Diagnostic{
		Severity:  DiagnosticWarning,
		Code:      code,
		Message:   message,
		TypePair:  typePair,
		FieldPath: fieldPath,
	})
}

// AddInfo adds an info diagnostic.
func (d *Diagnostics) AddInfo(code, message, typePair, fieldPath string) {
	d.Infos = append(d.Infos, Diagnostic{
		Severity:  DiagnosticInfo,
		Code:      code,
		Message:   message,
		TypePair:  typePair,
		FieldPath: fieldPath,
	})
}

// HasErrors returns true if there are any error diagnostics.
func (d *Diagnostics) HasErrors() bool {
	return len(d.Errors) > 0
}

// Merge merges another Diagnostics instance into this one.
func (d *Diagnostics) Merge(other Diagnostics) {
	d.Errors = append(d.Errors, other.Errors...)
	d.Warnings = append(d.Warnings, other.Warnings...)
	d.Infos = append(d.Infos, other.Infos...)
}

// IsValid returns true if there are no errors.
func (d *Diagnostics) IsValid() bool {
	return len(d.Errors) == 0
}

// Error returns a combined error from all error diagnostics, or nil if valid.
func (d *Diagnostics) Error() error {
	if d.IsValid() {
		return nil
	}

	var parts []string
	for _, e := range d.Errors {
		parts = append(parts, e.String())
	}

	return errors.New(strings.Join(parts, "; "))
}

// String returns a formatted diagnostic string.
func (d Diagnostic) String() string {
	var prefix []string
	if d.TypePair != "" {
		prefix = append(prefix, "["+d.TypePair+"]")
	}

	if d.FieldPath != "" {
		prefix = append(prefix, d.FieldPath)
	}

	msg := d.Message
	if d.Code != "" {
		msg = fmt.Sprintf("[%s] %s", d.Code, msg)
	}

	if len(prefix) > 0 {
		return strings.Join(prefix, " ") + ": " + msg
	}

	return msg
}
