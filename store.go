package main

import (
	"context"
)

// store defines the interface for storing and retrieving snippets
type store interface {
	// PutSnippet stores a new snippet with the given ID
	PutSnippet(ctx context.Context, id string, snippet *Snippet) error

	// GetSnippet retrieves a snippet by its ID
	GetSnippet(ctx context.Context, id string) (*Snippet, error)

	// DeleteSnippet removes a snippet by its ID
	DeleteSnippet(ctx context.Context, id string) error

	// UpdateSnippet modifies an existing snippet
	UpdateSnippet(ctx context.Context, id string, snippet *Snippet) error
}
