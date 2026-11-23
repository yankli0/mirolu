// Package engine compiles resource graphs into a deterministic DAG and runs Plan/Apply.
package engine

import (
	"errors"
	"fmt"

	"github.com/dominikbraun/graph"

	"github.com/yankli0/mirolu/internal/core"
)

var (
	ErrDuplicateResource = errors.New("duplicate resource id")
	ErrUnknownDependency = errors.New("unknown dependency resource id")
	ErrCycle             = errors.New("dependency cycle detected")
)

// DAG holds resources in deterministic topological execution order.
type DAG struct {
	resources map[string]core.Resource
	order     []string
}

// CompileDAG validates resources, detects duplicates/cycles, and sorts them deterministically.
func CompileDAG(resources []core.Resource) (*DAG, error) {
	g := graph.New(func(r core.Resource) string { return r.ID }, graph.Directed(), graph.Acyclic())
	resMap := make(map[string]core.Resource, len(resources))

	for _, res := range resources {
		if err := res.Validate(); err != nil {
			return nil, err
		}
		if _, exists := resMap[res.ID]; exists {
			return nil, fmt.Errorf("%w: %q", ErrDuplicateResource, res.ID)
		}
		resMap[res.ID] = res
		if err := g.AddVertex(res); err != nil {
			return nil, err
		}
	}

	for _, res := range resources {
		for _, req := range res.Requires {
			if err := addEdge(g, resMap, req, res.ID); err != nil {
				return nil, err
			}
		}
		for _, target := range res.Notifies {
			if err := addEdge(g, resMap, res.ID, target); err != nil {
				return nil, err
			}
		}
	}

	order, err := graph.StableTopologicalSort(g, func(a, b string) bool { return a < b })
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCycle, err)
	}

	return &DAG{resources: resMap, order: order}, nil
}

// Order returns resource IDs in topological execution order.
func (d *DAG) Order() []string {
	return d.order
}

// Resources returns resources in topological execution order.
func (d *DAG) Resources() []core.Resource {
	out := make([]core.Resource, 0, len(d.order))
	for _, id := range d.order {
		out = append(out, d.resources[id])
	}
	return out
}

func addEdge(g graph.Graph[string, core.Resource], resMap map[string]core.Resource, src, dst string) error {
	if _, ok := resMap[src]; !ok {
		return fmt.Errorf("%w: %q", ErrUnknownDependency, src)
	}
	if _, ok := resMap[dst]; !ok {
		return fmt.Errorf("%w: %q", ErrUnknownDependency, dst)
	}
	err := g.AddEdge(src, dst)
	switch {
	case err == nil, errors.Is(err, graph.ErrEdgeAlreadyExists):
		return nil
	case errors.Is(err, graph.ErrEdgeCreatesCycle):
		return fmt.Errorf("%w: %q -> %q", ErrCycle, src, dst)
	default:
		return err
	}
}
