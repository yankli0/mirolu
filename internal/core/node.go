// Package core defines Mirolu's four declarative concepts: Resource, Module,
// Node, and Secret.
package core

import "context"

// Node describes a target host managed by Mirolu.
type Node struct {
	Name    string   `json:"name"`
	Modules []Module `json:"-"`
}

// Module declaratively compiles system Resources for a given Node.
type Module interface {
	Name() string
	Build(ctx context.Context, node Node, secrets SecretLookup) ([]Resource, error)
}
