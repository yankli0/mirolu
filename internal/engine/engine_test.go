package engine_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/yankli0/mirolu/internal/core"
	"github.com/yankli0/mirolu/internal/engine"
)

type mockDriver struct {
	checkFn func(core.Resource) core.Diff
	applies int
	notifs  int
}

func (m *mockDriver) Check(_ context.Context, r core.Resource) (core.Diff, error) {
	if m.checkFn != nil {
		return m.checkFn(r), nil
	}
	return core.Diff{Action: core.ActionNoop}, nil
}

func (m *mockDriver) Apply(_ context.Context, _ core.Resource) error {
	m.applies++
	return nil
}

func (m *mockDriver) Notify(_ context.Context, _ core.Resource) error {
	m.notifs++
	return nil
}

type staticModule struct {
	name string
	res  []core.Resource
}

func (m staticModule) Name() string { return m.name }
func (m staticModule) Build(context.Context, core.Node, core.SecretLookup) ([]core.Resource, error) {
	return m.res, nil
}

func TestCompileDAG(t *testing.T) {
	dag, err := engine.CompileDAG([]core.Resource{
		{ID: "service:caddy", Module: "router", Kind: core.DriverService, Requires: []string{"file:cfg"}},
		{ID: "file:cfg", Module: "router", Kind: core.DriverFile, Notifies: []string{"service:caddy"}},
	})
	if err != nil {
		t.Fatalf("CompileDAG() error = %v", err)
	}
	if got, want := dag.Order(), []string{"file:cfg", "service:caddy"}; !slices.Equal(got, want) {
		t.Fatalf("Order() = %v, want %v", got, want)
	}

	errCases := []struct {
		name string
		res  []core.Resource
		want error
	}{
		{
			name: "duplicate id",
			res: []core.Resource{
				{ID: "a", Module: "m", Kind: core.DriverFile},
				{ID: "a", Module: "m", Kind: core.DriverFile},
			},
			want: engine.ErrDuplicateResource,
		},
		{
			name: "missing dependency",
			res: []core.Resource{
				{ID: "a", Module: "m", Kind: core.DriverFile, Requires: []string{"missing"}},
			},
			want: engine.ErrUnknownDependency,
		},
		{
			name: "cycle",
			res: []core.Resource{
				{ID: "a", Module: "m", Kind: core.DriverFile, Requires: []string{"b"}},
				{ID: "b", Module: "m", Kind: core.DriverFile, Requires: []string{"a"}},
			},
			want: engine.ErrCycle,
		},
	}

	for _, tc := range errCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := engine.CompileDAG(tc.res)
			if !errors.Is(err, tc.want) {
				t.Fatalf("CompileDAG() = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestPlanAndApplyNotifies(t *testing.T) {
	changed := true
	drv := &mockDriver{
		checkFn: func(r core.Resource) core.Diff {
			if changed && r.ID == "file:cfg" {
				return core.Diff{Changed: true, Action: core.ActionUpdate, Before: "old", After: "new", Sensitive: true}
			}
			return core.Diff{Action: core.ActionNoop}
		},
	}
	eng := engine.New(map[core.DriverKind]core.Driver{
		core.DriverFile:    drv,
		core.DriverService: drv,
	}, nil)

	node := core.Node{
		Name: "geko",
		Modules: []core.Module{staticModule{
			name: "router",
			res: []core.Resource{
				{ID: "file:cfg", Kind: core.DriverFile, Notifies: []string{"service:caddy"}},
				{ID: "service:caddy", Kind: core.DriverService},
			},
		}},
	}

	plan, err := eng.Plan(t.Context(), node)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if plan.ExitCode() != 2 || plan.ChangedCnt != 2 || plan.NotifiedCnt != 1 {
		t.Fatalf("unexpected plan counts: %+v", plan)
	}
	if drv.applies != 0 || drv.notifs != 0 {
		t.Fatalf("Plan mutated state: applies=%d notifs=%d", drv.applies, drv.notifs)
	}
	if plan.Changes[0].Diff.Before != "" || plan.Changes[0].Diff.After != "" {
		t.Fatalf("sensitive diff not cleared: %+v", plan.Changes[0].Diff)
	}

	if _, err := eng.Apply(t.Context(), node, true); err != nil {
		t.Fatalf("Apply(dryRun=true) error = %v", err)
	}
	if drv.applies != 0 || drv.notifs != 0 {
		t.Fatalf("DryRun mutated state: applies=%d notifs=%d", drv.applies, drv.notifs)
	}

	if _, err := eng.Apply(t.Context(), node, false); err != nil {
		t.Fatalf("Apply(dryRun=false) error = %v", err)
	}
	if drv.applies != 1 || drv.notifs != 1 {
		t.Fatalf("Apply counts = (applies=%d, notifs=%d), want (1, 1)", drv.applies, drv.notifs)
	}

	changed = false
	second, err := eng.Apply(t.Context(), node, false)
	if err != nil {
		t.Fatalf("second Apply() error = %v", err)
	}
	if second.ChangedCnt != 0 || second.UnchangedCnt != 2 {
		t.Fatalf("second Apply() not idempotent: %+v", second)
	}
}
