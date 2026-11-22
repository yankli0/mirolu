// Package ui renders Mirolu plans, unified diffs, and JSON output.
package ui

import (
	"fmt"
	"io"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/aymanbagabas/go-udiff"

	"github.com/yankli0/mirolu/internal/core"
)

// RenderOptions controls how plans and diffs are formatted.
type RenderOptions struct {
	Verbose bool // Include unchanged/noop rows when true.
	NoColor bool // Disable ANSI color escape sequences.
}

// UnifiedDiff computes a unified diff between before and after for path.
func UnifiedDiff(path, before, after string) string {
	if before == after {
		return ""
	}
	return udiff.Unified("a/"+path, "b/"+path, before, after)
}

// RenderPlan writes a hybrid compact Lipgloss v2 table, diffs, and summary footer.
func RenderPlan(w io.Writer, plan core.Plan, opts RenderOptions) error {
	var rows [][]string
	var diffs []core.Change

	for _, ch := range plan.Changes {
		if ch.Action.IsChange() || ch.Diff.Sensitive {
			diffs = append(diffs, ch)
		}
		if !opts.Verbose && !ch.Action.IsChange() && len(ch.NotifiedBy) == 0 {
			continue
		}
		details := ch.Diff.Summary
		if len(ch.NotifiedBy) > 0 {
			note := "notified by: " + strings.Join(ch.NotifiedBy, ", ")
			if details != "" {
				details += "; " + note
			} else {
				details = note
			}
		}
		rows = append(rows, []string{string(ch.Action), ch.ResourceID, ch.Module, string(ch.Kind), details})
	}

	if len(rows) == 0 {
		if _, err := fmt.Fprintf(w, "No changes. %s is up to date (%d unchanged).\n", plan.NodeName, plan.UnchangedCnt); err != nil {
			return err
		}
	} else {
		tbl := table.New().
			Border(lipgloss.NormalBorder()).
			BorderTop(false).
			BorderBottom(false).
			BorderLeft(false).
			BorderRight(false).
			BorderColumn(false).
			BorderHeader(true).
			Headers("ACTION", "RESOURCE", "MODULE", "KIND", "DETAILS").
			Rows(rows...).
			StyleFunc(func(row, _ int) lipgloss.Style {
				if opts.NoColor {
					return lipgloss.NewStyle()
				}
				if row == table.HeaderRow {
					return lipgloss.NewStyle().Bold(true).Padding(0, 1)
				}
				return lipgloss.NewStyle().Padding(0, 1)
			})
		if _, err := fmt.Fprintln(w, tbl.Render()); err != nil {
			return err
		}
	}

	for _, ch := range diffs {
		body := formatDiff(ch)
		if body == "" {
			continue
		}
		if _, err := fmt.Fprintf(w, "--- %s\n%s", ch.ResourceID, body); err != nil {
			return err
		}
		if !strings.HasSuffix(body, "\n") {
			if _, err := io.WriteString(w, "\n"); err != nil {
				return err
			}
		}
	}

	_, err := fmt.Fprintf(w, "Plan (%s): %d to change, %d to notify, %d unchanged.\n",
		plan.NodeName, plan.ChangedCnt, plan.NotifiedCnt, plan.UnchangedCnt)
	return err
}

func formatDiff(ch core.Change) string {
	if ch.Diff.Sensitive {
		return "[diff redacted: sensitive resource]\n"
	}
	if ch.Diff.UnifiedDiff != "" {
		return ch.Diff.UnifiedDiff
	}
	if ch.Diff.Before != "" || ch.Diff.After != "" {
		return UnifiedDiff(ch.ResourceID, ch.Diff.Before, ch.Diff.After)
	}
	return ""
}
