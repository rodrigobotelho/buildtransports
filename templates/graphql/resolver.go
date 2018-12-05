package graphql

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

//TODO funcs of the endpoint
