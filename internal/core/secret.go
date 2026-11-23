package core

import (
	"context"
	"log/slog"
)

// RedactedPlaceholder replaces secret bytes in formatted, logged, and JSON outputs.
const RedactedPlaceholder = "[REDACTED]"

// SecretRef identifies a secret requirement decoupled from storage backend.
type SecretRef struct {
	URI      string `json:"uri"`
	Required bool   `json:"required,omitempty"`
}

// SecretLookup resolves SecretRef URIs into SecretValue instances.
type SecretLookup interface {
	Resolve(ctx context.Context, ref SecretRef) (*SecretValue, error)
}

// SecretValue is an opaque, redacted secret handle (expanded in Stage 5).
type SecretValue struct {
	raw []byte //nolint:unused // populated in Stage 5
}

func (SecretValue) String() string       { return RedactedPlaceholder }
func (SecretValue) LogValue() slog.Value { return slog.StringValue(RedactedPlaceholder) }
func (SecretValue) MarshalJSON() ([]byte, error) {
	return []byte(`"` + RedactedPlaceholder + `"`), nil
}
