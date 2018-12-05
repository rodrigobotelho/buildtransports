package main

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/tools/go/ast/astutil"

	_ "google.golang.org/grpc"
)

const pathPrefixSrc = `
//PathPrefix Prefixo do caminho do serviço
const PathPrefix=""`
const graphqlAddr = `
var graphqlAddr = fs.String("graphql-addr", ":8084", "graphql listen address")`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(
			os.Stderr,
			"Uso: %v <nome de servico>\n",
			filepath.Base(os.Args[0]),
		)
		os.Exit(1)
	}
	for _, tool := range []string{
		"github.com/kujtimiihoxha/kit",
		"golang.org/x/tools/cmd/goimports",
		"github.com/ksubedi/gomove",
		"github.com/golang/protobuf/protoc-gen-go",
	} {
		cmd := tool[strings.LastIndex(tool, "/")+1:]
		if _, err := exec.LookPath(cmd); err != nil {
			run("go get " + tool)
		}
	}
	serv := os.Args[1]
	service := serv + "/cmd/service/service.go"
	servName := strings.Title(serv)
	httpHandler := serv + "/pkg/http/handler.go"
	templates := build.Default.GOPATH +
		"/src/github.com/rodrigobotelho/buildtransports/templates"
	if !servicoJaExiste(serv) {
		run("kit n s " + serv)
		appendTo(serv+"/pkg/service/service.go", pathPrefixSrc)
		fmt.Println("Adicione os métodos que serão utilizados no serviço: " +
			"pkg/service/service.go")
		os.Exit(0)
	}
	corrigePastas(serv)
	defer func() {
		if r := recover(); r != nil {
			panic(r)
		}
		fmt.Println("Corrigindo pastas...")
		err := os.MkdirAll(serv+"/pkg/apis", os.ModePerm)
		check(err, "erro ao criar pasta pkg/apis: %v", err)
		corrigePastas(serv)
		run("goimports -w %v/cmd/service/init_service.go", serv)
		run("goimports -w %v/pkg/apis/service/middleware.go", serv)
		run("goimports -w %v/pkg/apis/endpoint/endpoint.go", serv)
	}()
	for {
		fmt.Println("Indique qual transporte: http, grpc, graphql " +
			"(ou vazio para encerrar)")
		var transporte string
		fmt.Scanln(&transporte)
		if transporte == "" {
			break
		}
		if transporte == "graphql" {
			sn := "s"
			if _, err := os.Stat(serv + "/pkg/apis/graphql"); !os.IsNotExist(err) {
				fmt.Println("Transporte graphql já existente, " +
					"TEM CERTEZA que deseja substituí-lo? s ou n?")
				fmt.Scanln(&sn)
			}
			if sn == "s" {
				criaEstruturaDePastasBasicasSeNecessario(serv)
				resolver := serv + "/pkg/apis/graphql/resolver.go"
				handler := serv + "/pkg/apis/graphql/handler.go"
				handlerTest := serv + "/pkg/apis/graphql/handler_test.go"
				schema := serv + "/pkg/apis/graphql/schema.graphql"
				err := os.MkdirAll(serv+"/pkg/apis/graphql", os.ModePerm)
				check(err, "erro ao criar pasta: '%v'", err)
				_, err = os.OpenFile(schema, os.O_RDONLY|os.O_CREATE, os.ModePerm)
				check(err, "erro ao criar schema: %v", err)
				for _, file := range []string{resolver, handler, handlerTest} {
					name := file[strings.LastIndex(file, "/"):]
					cp(templates+"/graphql/"+name, file)
					sed(file, "Example", servName)
					sed(file, "example", serv)
					run("goimports -w " + file)
				}
				b1, err := ioutil.ReadFile(templates + "/graphql/init_handler.go")
				check(err, "erro ao ler arquivo: %v", err)
				initHandler := string(b1)
				b2, err := ioutil.ReadFile(service)
				check(err, "erro ao ler arquivo: %v", err)
				if !strings.Contains(string(b2), "http1") {
					initHandler = strings.Replace(initHandler, "http1", "http", -1)
				}
				appendTo(service, initHandler)
				sed(service, "Example", servName)
				sed(service, "example", serv)
				sed(service, "var grpcAddr.*", "&"+graphqlAddr)
				sed(
					service,
					"g := createService.*",
					"&\n\tinitGraphqlHandler(svc, g)",
				)
				run("goimports -w " + service)
			}
		} else {
			fmt.Println("Indique os métodos separados por espaço, " +
				"vazio se todos.")
			var metodos string
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				metodos = scanner.Text()
				if metodos != "" {
					metodos = "-m " + strings.Join(strings.Fields(metodos), " -m ")
				}
			}
			if transporte != "" {
				transporte = "-t " + transporte
			}
			run(
				"kit g s %v --endpoint-mdw --svc-mdw %v %v",
				serv,
				transporte,
				metodos,
			)
			run("kit g c %v %v", serv, transporte)
			if _, err := os.Stat(httpHandler); !os.IsNotExist(err) {
				b, err := ioutil.ReadFile(httpHandler)
				check(err, "erro ao ler arquivo '%v': %v", httpHandler, err)
				if !strings.Contains(string(b), "PathPrefix") {
					sed(httpHandler, `\(\"\/.*\"\)`, "service.PathPrefix")
					run("goimports -w " + httpHandler)
				}
			}
		}
		in := templates + "/init_service.go"
		out := serv + "/cmd/service/init_service.go"
		cp(in, out)
		sed(out, "Example", servName)
		sed(service, "svc := service.New.*", "svc := initService()")
	}
}

func criaEstruturaDePastasBasicasSeNecessario(serv string) {
	if _, err := os.Stat(serv + "/cmd/service/service.go"); !os.IsNotExist(err) {
		return
	}
	run("kit g s %v --endpoint-mdw --svc-mdw", serv)
	run("kit g c %v", serv)
	err := os.RemoveAll(serv + "/pkg/http")
	check(err, "erro ao remover pasta: %v", err)
	err = os.RemoveAll(serv + "/client/http")
	check(err, "erro ao remover pasta: %v", err)
	deleteFunc(serv+"/cmd/service/service.go", "initHttpHandler")
	deleteFunc(serv+"/cmd/service/service_gen.go", "defaultHttpOptions")
	sed(serv+"/cmd/service/service_gen.go", ".*initHttpHandler.*", "")
}

func servicoJaExiste(serv string) bool {
	for _, p := range []string{"/pkg/service", "/pkg/apis/service"} {
		if _, err := os.Stat(serv + p + "/service.go"); !os.IsNotExist(err) {
			return true
		}
	}
	return false
}

func corrigePastas(serv string) {
	if _, err := os.Stat(serv + "/pkg/apis"); os.IsNotExist(err) {
		return
	}
	move(serv, "service")
	move(serv, "grpc")
	move(serv, "http")
	move(serv, "endpoint")
}

func move(serv, path string) {
	_, err1 := os.Stat(fmt.Sprintf("%v/pkg/%v", serv, path))
	_, err2 := os.Stat(fmt.Sprintf("%v/pkg/apis/%v", serv, path))
	if !os.IsNotExist(err1) && os.IsNotExist(err2) {
		err := os.Rename(serv+"/pkg/"+path, serv+"/pkg/apis/"+path)
		check(err, "erro ao mover pastas: %v", err)
		run("gomove %v/pkg/%v %v/pkg/apis/%v", serv, path, serv, path)
	} else if os.IsNotExist(err1) && !os.IsNotExist(err2) {
		err := os.Rename(serv+"/pkg/apis/"+path, serv+"/pkg/"+path)
		check(err, "erro ao mover pastas: %v", err)
		run("gomove %v/pkg/apis/%v %v/pkg/%v", serv, path, serv, path)
	}
}

func run(format string, a ...interface{}) string {
	cmd := fmt.Sprintf(format, a...)
	args := strings.Fields(cmd)
	b, err := exec.Command(args[0], args[1:]...).CombinedOutput()
	out := strings.TrimSpace(string(b))
	check(err, "erro ao executar '%v': %v", cmd, out)
	return string(b)
}

func appendTo(file, text string) {
	f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0644)
	check(err, "erro ao abrir arquivo '%v': %v", f, err)
	_, err = f.WriteString(text)
	check(err, "erro ao escrever no arquivo '%v': %v", f, err)
}

func sed(file, old, new string) {
	b, err := ioutil.ReadFile(file)
	check(err, "erro ao ler arquivo '%v': %v", file, err)
	re := regexp.MustCompile(old)
	var str string
	if len(new) > 0 && new[0] == '&' {
		str = re.ReplaceAllStringFunc(string(b), func(s string) string {
			return s + new[1:]
		})
	} else {
		str = re.ReplaceAllString(string(b), new)
	}
	err = ioutil.WriteFile(file, []byte(str), os.ModePerm)
	check(err, "erro ao escrever arquivo '%v': %v", file, err)
}

func cp(in, out string) {
	b, err := ioutil.ReadFile(in)
	check(err, "erro ao ler arquivo '%v': %v", in, err)
	err = ioutil.WriteFile(out, b, os.ModePerm)
	check(err, "erro ao escrever arquivo '%v': %v", out, err)
}

func deleteFunc(path, fn string) {
	af := func(c *astutil.Cursor) bool {
		n, ok := c.Node().(*ast.FuncDecl)
		if !ok {
			return true
		}
		if n.Name.Name == fn {
			c.Delete()
			return false
		}
		return true
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	check(err, "erro ao fazer parse de '%v': %v", path, err)
	result := astutil.Apply(file, af, nil)
	var buf bytes.Buffer
	err = printer.Fprint(&buf, fset, result)
	check(err, "erro ao imprimir '%v': %v", path, err)
	err = ioutil.WriteFile(path, buf.Bytes(), os.ModePerm)
	check(err, "erro ao gravar '%v': %v", path, err)
	run("goimports -w " + path)
}

func check(err error, msg string, a ...interface{}) {
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf(msg, a...))
		os.Exit(1)
	}
}
