package grpc

import (
	"dbtesting"
	"net"
	"os"
	"strings"
	"testing"

	endpoint1 "github.com/go-kit/kit/endpoint"
	"github.com/mcesar/dbrx"
	grpc1 "google.golang.org/grpc"
)

func TestBuscar(t *testing.T) {
	casos := []struct {
		nome    string
		script  string
		wantErr bool
	}{}
	for _, caso := range casos {
		t.Run(caso.nome, grpcRun(caso.script, func(client service.ExemploService, dml dbrx.DML, t *testing.T) {
			// TODO: Executar método(s) do client e verificar o resultado
		}))
	}
}

func grpcRun(script string, tf func(service.ExemploService, dbrx.DML, *testing.T)) func(*testing.T) {
	return func(t *testing.T) {
		f, _ := os.Open("../../schema.sql")
		_, conn := dbtesting.ExecScripts(f, strings.NewReader(script))
		svc := service.NewBasicExemploService(conn)
		endpoints := endpoint.New(svc, map[string][]endpoint1.Middleware{})
		grpcServer := NewGRPCServer(endpoints, nil)
		grpcListener, err := net.Listen("tcp", ":8888")
		if err != nil {
			panic(err)
		}
		baseServer := grpc1.NewServer()
		defer baseServer.GracefulStop()
		defer grpcListener.Close()
		go func() error {
			pb.RegisterExemploServer(baseServer, grpcServer)
			return baseServer.Serve(grpcListener)
		}()
		cc, err := grpc1.Dial(":8888", grpc1.WithInsecure())
		if err != nil {
			panic(err)
		}
		client, err := grpc.New(cc, nil)
		if err != nil {
			panic(err)
		}
		tf(client, dbrx.Wrap(conn.NewSession(nil)), t)
	}
}
