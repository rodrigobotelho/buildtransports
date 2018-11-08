package graphql

import (
	"context"
)

// Resolver Resolve o schema.graphql
type Resolver struct {
	example service.ExampleService
}

// NewResolver cria um resolver graphql
func NewResolver(example service.ExampleService) *Resolver {
	return &Resolver{
		example: example,
	}
}

//Args Graphql parameters
type Args struct {
}

//TODO funcs of the endpoint
