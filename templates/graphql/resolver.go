package graphql

import (
	"context"
)

type Resolver struct {
	example service.ExampleService
}

func NewResolver(example service.ExampleService) *Resolver {
	return &Resolver{
		example: example,
	}
}

//Args Graphql parameters
type Args struct {
}

//TODO funcs of the endpoint
