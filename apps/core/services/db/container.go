package db

import (
	"context"
	"sync"
)

type Container struct {
	Postgres Repository
	Scylla   Repository
}

var (
	container     *Container
	containerOnce sync.Once
)

type contextKey string

const ContainerKey contextKey = "container"

// InitializeContainer initializes the container with the repositories
func InitializeContainer(pgRepo, scyllaRepo Repository) {
	containerOnce.Do(func() {
		container = &Container{
			Postgres: pgRepo,
			Scylla:   scyllaRepo,
		}
	})
}

// GetContainer returns the singleton container
func GetContainer() *Container {
	if container == nil {
		panic("Container not initialized. Call InitializeContainer() first.")
	}
	return container
}

// WithContainerContext adds the container to the context
func WithContainerContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, ContainerKey, GetContainer())
}

// FromContext retrieves the container from the context
func FromContext(ctx context.Context) *Container {
	container, ok := ctx.Value("container").(*Container)
	if !ok {
		panic("Container not found in context")
	}
	return container
}
