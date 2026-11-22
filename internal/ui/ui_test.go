package ui_test

import (
	"bytes"
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/yankli0/mirolu/internal/core"
	"github.com/yankli0/mirolu/internal/ui"
)

func TestRenderPlan(t *testing.T) {
	plan := core.Plan{
		NodeName:     "geko",
		ChangedCnt:   2,
		NotifiedCnt:  1,
		UnchangedCnt: 1,
		Changes: []core.Change{
			{ResourceID: "file:/etc/motd", Module: "baseos", Kind: core.DriverFile, Action: core.ActionNoop},
			{
				ResourceID: "file:/etc/caddy/Caddyfile",
				Module:     "router",
				Kind:       core.DriverFile,
				Action:     core.ActionUpdate,
				Diff:       core.Diff{Changed: true, Action: core.ActionUpdate, Before: "a\n", After: "b\n"},
			},
			{
				ResourceID: "service:caddy",
				Module:     "router",
				Kind:       core.DriverService,
				Action:     core.ActionNotify,
				NotifiedBy: []string{"file:/etc/caddy/Caddyfile"},
			},
		},
	}

	var compact bytes.Buffer
	if err := ui.RenderPlan(&compact, plan, ui.RenderOptions{NoColor: true}); err != nil {
		t.Fatalf("RenderPlan() error = %v", err)
	}
	got := compact.String()
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("NoColor output contains ANSI escapes: %q", got)
	}
	if strings.Contains(got, "/etc/motd") {
		t.Fatalf("compact output should hide unchanged row: %s", got)
	}
	for _, want := range []string{"file:/etc/caddy/Caddyfile", "-a", "+b", "Plan (geko): 2 to change, 1 to notify, 1 unchanged."} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}

	var verbose bytes.Buffer
	if err := ui.RenderPlan(&verbose, plan, ui.RenderOptions{Verbose: true, NoColor: true}); err != nil {
		t.Fatalf("RenderPlan(Verbose) error = %v", err)
	}
	if !strings.Contains(verbose.String(), "/etc/motd") {
		t.Fatalf("verbose output missing unchanged row:\n%s", verbose.String())
	}
}

func TestWriteJSON(t *testing.T) {
	var buf bytes.Buffer
	payload := struct {
		Plan   core.Plan        `json:"plan"`
		Secret core.SecretValue `json:"secret"`
	}{
		Plan: core.Plan{NodeName: "geko", UnchangedCnt: 1},
	}

	if err := ui.WriteJSON(&buf, payload); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	var decoded struct {
		Plan   core.Plan `json:"plan"`
		Secret string    `json:"secret"`
	}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if decoded.Plan.NodeName != "geko" || decoded.Secret != core.RedactedPlaceholder {
		t.Fatalf("decoded = %+v", decoded)
	}
}
