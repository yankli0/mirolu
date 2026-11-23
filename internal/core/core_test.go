package core_test

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/yankli0/mirolu/internal/core"
)

func TestResourceValidate(t *testing.T) {
	tests := []struct {
		name    string
		res     core.Resource
		wantErr bool
	}{
		{
			name: "valid",
			res:  core.Resource{ID: "file:/etc/motd", Module: "baseos", Kind: core.DriverFile},
		},
		{
			name:    "empty id",
			res:     core.Resource{Module: "baseos", Kind: core.DriverFile},
			wantErr: true,
		},
		{
			name:    "invalid kind",
			res:     core.Resource{ID: "x", Module: "baseos", Kind: "unknown"},
			wantErr: true,
		},
		{
			name:    "empty requires",
			res:     core.Resource{ID: "x", Module: "baseos", Kind: core.DriverFile, Requires: []string{""}},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.res.Validate()
			if tc.wantErr && !errors.Is(err, core.ErrInvalidResource) {
				t.Fatalf("Validate() = %v, want ErrInvalidResource", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("Validate() unexpected error: %v", err)
			}
		})
	}
}

func TestPlanAndSecretRedaction(t *testing.T) {
	if (core.Plan{ChangedCnt: 0}).ExitCode() != 0 {
		t.Fatal("empty plan ExitCode() != 0")
	}
	if (core.Plan{ChangedCnt: 1}).ExitCode() != 2 {
		t.Fatal("changed plan ExitCode() != 2")
	}

	var sec core.SecretValue
	if got := fmt.Sprint(sec); got != core.RedactedPlaceholder {
		t.Fatalf("fmt.Sprint(SecretValue) = %q, want %q", got, core.RedactedPlaceholder)
	}

	var logBuf bytes.Buffer
	slog.New(slog.NewTextHandler(&logBuf, nil)).InfoContext(t.Context(), "secret check", "secret", sec)
	if !strings.Contains(logBuf.String(), core.RedactedPlaceholder) {
		t.Fatalf("slog output missing placeholder: %q", logBuf.String())
	}

	raw, err := json.Marshal(sec)
	if err != nil {
		t.Fatalf("json.Marshal(SecretValue) error = %v", err)
	}
	if string(raw) != `"`+core.RedactedPlaceholder+`"` {
		t.Fatalf("json.Marshal(SecretValue) = %s, want %q", raw, core.RedactedPlaceholder)
	}
}
