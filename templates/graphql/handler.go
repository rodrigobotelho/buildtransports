package graphql

import (
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-kit/kit/log"
	graphqlkit "github.com/rodrigobotelho/graphql-kit"
)

type GraphqlHandler struct {
	example     service.ExampleService
	secret      string
	Schema      string
	logger      log.Logger
}

func NewHandler(example service.ExampleService, schema, secret string, logger log.Logger) *GraphqlHandler {
	return &GraphqlHandler{
		example:     example,
		secret:      secret,
		Schema:      schema,
		logger:      logger,
	}
}

func (h *GraphqlHandler) Handler() http.Handler {
	res := NewResolver(h.example)
	handler := graphqlkit.Handlers{}
    handler.AddFullGraphqlService(
		h.Schema,
		res,
		h.logger,
		"siop",
		"module_name",
		h.secret,
		jwt.SigningMethodHS512,
		&jwt.StandardClaims{},
	)
	return handler.Handler()
}

