package graphql

import (
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-kit/kit/log"
	graphqlkit "github.com/rodrigobotelho/graphql-kit"
)

// Handler Usado para criar um servidor graphql com autenticação
type Handler struct {
	example service.ExampleService
	secret  string
	Schema  string
	logger  log.Logger
}

// NewHandler Cria um novo handler graphql com autenticação e logging
func NewHandler(example service.ExampleService, schema, secret string, logger log.Logger) *Handler {
	return &Handler{
		example: example,
		secret:  secret,
		Schema:  schema,
		logger:  logger,
	}
}

// Handler Retorna um handler http que vai cuidar de requisições graphql
func (h *Handler) Handler() http.Handler {
	res := NewResolver(h.example)
	handlers := graphqlkit.Handlers{}
	handlers.AddFullGraphqlService(
		h.Schema,
		res,
		h.logger,
		"siop",
		"module_name",
		h.secret,
		jwt.SigningMethodHS512,
		&jwt.StandardClaims{},
	)
	return handlers.Handler()
}
