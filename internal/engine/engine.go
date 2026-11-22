package engine

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/yankli0/mirolu/internal/core"
)

// ErrMissingDriver indicates no Driver was registered for a resource's DriverKind.
var ErrMissingDriver = errors.New("no driver registered for resource kind")

// Engine compiles node modules into a DAG and executes Plan and Apply runs.
type Engine struct {
	drivers map[core.DriverKind]core.Driver
	secrets core.SecretLookup
}

// New constructs an Engine with the given drivers and secret lookup.
func New(drivers map[core.DriverKind]core.Driver, secrets core.SecretLookup) *Engine {
	return &Engine{drivers: drivers, secrets: secrets}
}

// Plan compiles node and performs a read-only Check pass across the DAG.
func (e *Engine) Plan(ctx context.Context, node core.Node) (core.Plan, error) {
	return e.run(ctx, node, false)
}

// Apply compiles node and converges state unless dryRun is true.
func (e *Engine) Apply(ctx context.Context, node core.Node, dryRun bool) (core.Plan, error) {
	return e.run(ctx, node, !dryRun)
}

func (e *Engine) run(ctx context.Context, node core.Node, apply bool) (core.Plan, error) {
	var all []core.Resource
	for _, mod := range node.Modules {
		built, err := mod.Build(ctx, node, e.secrets)
		if err != nil {
			return core.Plan{}, fmt.Errorf("module %q: %w", mod.Name(), err)
		}
		for i := range built {
			if built[i].Module == "" {
				built[i].Module = mod.Name()
			}
		}
		all = append(all, built...)
	}

	dag, err := CompileDAG(all)
	if err != nil {
		return core.Plan{}, err
	}

	resources := dag.Resources()
	plan := core.Plan{
		NodeName: node.Name,
		Order:    dag.Order(),
		Changes:  make([]core.Change, 0, len(resources)),
	}

	pendingNotifies := make(map[string][]string)
	for _, res := range resources {
		if err := ctx.Err(); err != nil {
			return plan, err
		}

		drv := e.drivers[res.Kind]
		if drv == nil {
			return plan, fmt.Errorf("%w: %q", ErrMissingDriver, res.Kind)
		}

		diff, err := drv.Check(ctx, res)
		if err != nil {
			return plan, fmt.Errorf("check %q: %w", res.ID, err)
		}

		notifiers := uniqueSorted(pendingNotifies[res.ID])
		if apply {
			if diff.Changed {
				if err := drv.Apply(ctx, res); err != nil {
					return plan, fmt.Errorf("apply %q: %w", res.ID, err)
				}
			}
			if len(notifiers) > 0 {
				if err := drv.Notify(ctx, res); err != nil {
					return plan, fmt.Errorf("notify %q: %w", res.ID, err)
				}
			}
		}

		action := resolveAction(diff, notifiers)
		if diff.Changed || len(notifiers) > 0 {
			for _, target := range res.Notifies {
				pendingNotifies[target] = append(pendingNotifies[target], res.ID)
			}
		}

		if diff.Sensitive {
			diff.Before, diff.After, diff.UnifiedDiff = "", "", ""
		}
		diff.Action = action

		plan.Changes = append(plan.Changes, core.Change{
			ResourceID: res.ID,
			Module:     res.Module,
			Kind:       res.Kind,
			Action:     action,
			NotifiedBy: notifiers,
			Diff:       diff,
		})

		if action.IsChange() {
			plan.ChangedCnt++
		} else {
			plan.UnchangedCnt++
		}
		if len(notifiers) > 0 {
			plan.NotifiedCnt++
		}
	}

	return plan, nil
}

func resolveAction(diff core.Diff, notifiers []string) core.Action {
	switch {
	case diff.Changed && diff.Action.IsChange():
		return diff.Action
	case diff.Changed:
		return core.ActionUpdate
	case len(notifiers) > 0:
		return core.ActionNotify
	default:
		return core.ActionNoop
	}
}

func uniqueSorted(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := slices.Clone(items)
	slices.Sort(out)
	return slices.Compact(out)
}
