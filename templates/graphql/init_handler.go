func initGraphqlHandler(svc service.ExampleService, g *group.Group) {
    //Mude aqui o schema
    schema:="pkg/graphql/schema.graphql"
    secret:=os.Getenv("AUTH_PRIVATE_KEY")
	graphqlHandler := graphql.NewHandler(
		svc,
		schema,
		secret,
		logger,
	)
	mux := http1.NewServeMux()

	mux.Handle("/graphql", graphqlHandler.Handler())
	httplistener, err := net.Listen("tcp", *graphqlAddr)
	if err != nil {
		logger.Log("transport", "http", "during", "listen", "err", err)
	} else {
		g.Add(func() error {
			logger.Log("transport", "graphql", "addr", *graphqlAddr)
			return http1.Serve(httplistener, mux)
		}, func(error) {
			httplistener.Close()
		})
	}
}
