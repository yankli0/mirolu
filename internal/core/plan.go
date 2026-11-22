package core

import "context"

// Action classifies the planned transition for a resource.
type Action string

const (
	ActionNoop   Action = "noop"
	ActionCreate Action = "create"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
	ActionNotify Action = "notify"
)

// IsChange reports whether the action modifies or notifies the target resource.
func (a Action) IsChange() bool {
	return a != ActionNoop && a != ""
}

// Diff captures the difference between observed and desired resource state.
type Diff struct {
	Changed     bool   `json:"changed"`
	Action      Action `json:"action"`
	Summary     string `json:"summary,omitempty"`
	Before      string `json:"before,omitempty"`
	After       string `json:"after,omitempty"`
	UnifiedDiff string `json:"unified_diff,omitempty"`
	Sensitive   bool   `json:"sensitive,omitempty"`
}

// Change pairs a resource identity with its planned action, notification sources, and diff.
type Change struct {
	ResourceID string     `json:"resource_id"`
	Module     string     `json:"module"`
	Kind       DriverKind `json:"kind"`
	Action     Action     `json:"action"`
	NotifiedBy []string   `json:"notified_by,omitempty"`
	Diff       Diff       `json:"diff"`
}

// Plan represents a deterministic read-only execution plan for a single Node.
type Plan struct {
	NodeName     string   `json:"node_name"`
	Order        []string `json:"order"`
	Changes      []Change `json:"changes"`
	ChangedCnt   int      `json:"changed_cnt"`
	NotifiedCnt  int      `json:"notified_cnt"`
	UnchangedCnt int      `json:"unchanged_cnt"`
}

// HasChanges reports whether the plan contains any pending changes.
func (p Plan) HasChanges() bool { return p.ChangedCnt > 0 }

// ExitCode returns 2 when changes are pending and 0 when up to date.
func (p Plan) ExitCode() int {
	if p.HasChanges() {
		return 2
	}
	return 0
}

// Driver implements idempotent state inspection, application, and notification.
type Driver interface {
	Check(ctx context.Context, res Resource) (Diff, error)
	Apply(ctx context.Context, res Resource) error
	Notify(ctx context.Context, res Resource) error
}
