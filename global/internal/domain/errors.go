package domain

import "errors"

// ErrNotFound is the domain-level absence error. Driven adapters translate
// their technology-specific errors (for example pgx.ErrNoRows) into it at the
// boundary so the core and driving adapters never import persistence packages
// to classify errors.
var ErrNotFound = errors.New("not found")

// IsNotFound reports whether err represents a missing record.
func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }
