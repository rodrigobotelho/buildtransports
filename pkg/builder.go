package builder

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
	"regexp"
	"strings"

	"golang.org/x/tools/go/ast/astutil"

	_ "github.com/graph-gophers/graphql-go" // just to make sure it is available
	_ "github.com/mcesar/copier"
	_ "github.com/rodrigobotelho/graphql-kit"
	_ "google.golang.org/grpc"
)

const pathPrefixSrc = `
//PathPrefix Prefixo do caminho do serviço
const PathPrefix=""`
const graphqlAddr = `
var graphqlAddr = fs.String("graphql-addr", ":8084", "graphql listen address")`
const graphqlGoPath = "src/github.com/graph-gophers/graphql-go"
const patchGraphqlKit = `opts := []graphql.SchemaOpt{graphql.UseFieldResolvers()}
	   return graphql.MustParseSchema(string(schemaFile), res, opts...)`
const patchDockerfile1 = `RUN git clone https://github.com/salman-ahmad/graphql-go /patch \
&& cd /patch \
&& git format-patch \
	940d2b01f2549ee5d9e87141ea909134bf56e3a9..\
	86130ac51668b74fefdb5fca5cf78a8865a26845 \
&& cd /go/pkg/mod/github.com/graph-gophers/$(ls /go/pkg/mod/github.com/graph-gophers) \
&& git apply /patch/*.patch \
&& cd /app`
const patchDockerfile2 = `RUN sed -i 's/return graphql.MustParseSchema.*/opts := []graphql.SchemaOpt{graphql.UseFieldResolvers()}\n        return graphql.MustParseSchema(string(schemaFile), res, opts...)/g' \
	/go/pkg/mod/github.com/rodrigobotelho/$(ls /go/pkg/mod/github.com/rodrigobotelho)/service.go`
const schemaGraphqlDockerfile = "COPY --from=stage1 /app/pkg/apis/graphql/schema.graphql /pkg/apis/graphql/schema.graphql"

// Build builds the transports
func Build(serv string) {
	for _, tool := range []string{
		"github.com/kujtimiihoxha/kit",
		"golang.org/x/tools/cmd/goimports",
		"github.com/ksubedi/gomove",
		"github.com/golang/protobuf/protoc-gen-go",
	} {
		cmd := tool[strings.LastIndex(tool, "/")+1:]
		if _, err := exec.LookPath(cmd); err != nil {
			Run("go get " + tool)
		}
	}
	service := serv + "/cmd/service/service.go"
	servName := strings.Title(serv)
	httpHandler := serv + "/pkg/http/handler.go"
	templates := build.Default.GOPATH +
		"/src/github.com/rodrigobotelho/buildtransports/templates"
	if !servicoJaExiste(serv) {
		Run("kit n s " + serv)
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
		Run("goimports -w %v/cmd/service/init_service.go", serv)
		Run("goimports -w %v/pkg/apis/service/middleware.go", serv)
		Run("goimports -w %v/pkg/apis/endpoint/endpoint.go", serv)
	}()
	count := 0
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
			if sn == "n" {
				continue
			}
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
				Sed(file, "Example", servName)
				Sed(file, "example", serv)
				Run("goimports -w " + file)
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
			Sed(service, "Example", servName)
			Sed(service, "example", serv)
			Sed(service, "var grpcAddr.*", "&"+graphqlAddr)
			Sed(
				service,
				"g := createService.*",
				"&\n\tinitGraphqlHandler(svc, g)",
			)
			Sed(
				handler,
				`"module_name"`,
				`fmt.Sprintf("`+servName+`_%d", time.Now().Nanosecond())`,
			)
			Run("goimports -w " + service)
			Run("goimports -w " + handler)
			aplicaPatchDoGraphqlGo()
			count++
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
			Run(
				"kit g s %v --endpoint-mdw --svc-mdw %v %v",
				serv,
				transporte,
				metodos,
			)
			Run("kit g c %v %v", serv, transporte)
			if _, err := os.Stat(httpHandler); !os.IsNotExist(err) {
				b, err := ioutil.ReadFile(httpHandler)
				check(err, "erro ao ler arquivo '%v': %v", httpHandler, err)
				if !strings.Contains(string(b), "PathPrefix") {
					Sed(httpHandler, `\(\"\/.*\"\)`, "service.PathPrefix")
					Run("goimports -w " + httpHandler)
				}
			}
			if transporte == "-t grpc" {
				replaceGprcEncodersAndDecoders(serv + "/pkg/grpc/handler.go")
				replaceGprcEncodersAndDecoders(serv + "/client/grpc/grpc.go")
				handlerTest := serv + "/pkg/grpc/handler_test.go"
				if _, err := os.Stat(handlerTest); os.IsNotExist(err) {
					cp(templates+"/grpc/handler_test.go", handlerTest)
					Sed(handlerTest, "Exemplo", servName)
					Run("goimports -w " + handlerTest)
				}
			}
			count++
		}
	}
	if count == 0 {
		return
	}

	if _, err := os.Stat(serv + "/cmd/service/init_service.go"); os.IsNotExist(err) {
		in := templates + "/init_service.go"
		out := serv + "/cmd/service/init_service.go"
		cp(in, out)
		Sed(out, "Example", servName)
	}
	Sed(service, "svc := service.New.*", "svc := initService()")
	Sed(service, "func Run.*", "// Run runs service\n&")
	Sed(serv+"/pkg/endpoint/endpoint.go", "// Failer", "// Failure")
	if _, err := os.Stat(serv + "/Dockerfile"); os.IsNotExist(err) {
		cp(templates+"/Dockerfile", serv+"/Dockerfile")
		if _, err := os.Stat(serv + "/pkg/apis/graphql"); !os.IsNotExist(err) {
			Sed(serv+"/Dockerfile", "exemplo", servName)
			Sed(serv+"/Dockerfile", "# graphql patch1", patchDockerfile1)
			Sed(serv+"/Dockerfile", "# graphql patch2", patchDockerfile2)
			Sed(
				serv+"/Dockerfile",
				"# copy schema.graphql",
				schemaGraphqlDockerfile,
			)
		}
	}
	Sed(service, "var svc .*= NewBasic", "var svc = NewBasic")
}

func criaEstruturaDePastasBasicasSeNecessario(serv string) {
	if _, err := os.Stat(serv + "/cmd/service/service.go"); !os.IsNotExist(err) {
		return
	}
	Run("kit g s %v --endpoint-mdw --svc-mdw", serv)
	Run("kit g c %v", serv)
	err := os.RemoveAll(serv + "/pkg/http")
	check(err, "erro ao remover pasta: %v", err)
	err = os.RemoveAll(serv + "/client/http")
	check(err, "erro ao remover pasta: %v", err)
	deleteFunc(serv+"/cmd/service/service.go", "initHttpHandler")
	deleteFunc(serv+"/cmd/service/service_gen.go", "defaultHttpOptions")
	Sed(serv+"/cmd/service/service_gen.go", ".*initHttpHandler.*", "")
	Sed(serv+"/pkg/service/service.go", "var svc .* =", "var svc =")
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
		Run("gomove %v/pkg/%v %v/pkg/apis/%v", serv, path, serv, path)
	} else if os.IsNotExist(err1) && !os.IsNotExist(err2) {
		err := os.Rename(serv+"/pkg/apis/"+path, serv+"/pkg/"+path)
		check(err, "erro ao mover pastas: %v", err)
		Run("gomove %v/pkg/apis/%v %v/pkg/%v", serv, path, serv, path)
	}
}

func aplicaPatchDoGraphqlGo() {
	if graphqlGoPermiteUsarCamposNoResolver() {
		return
	}
	Run("go get -u github.com/graph-gophers/graphql-go")
	if graphqlGoPermiteUsarCamposNoResolver() {
		return
	}
	gopath := build.Default.GOPATH
	patch := "src/github.com/rodrigobotelho/buildtransports/patches/" +
		"0001-Use-struct-fields-as-resolvers-instead-of-methods-28.patch"
	dir, err := os.Getwd()
	check(err, "erro ao obter o diretório atual: %v", err)
	os.Chdir(gopath + "/" + graphqlGoPath)
	Run("git apply " + gopath + "/" + patch)
	os.Chdir(dir)
	Sed(
		gopath+"/src/github.com/rodrigobotelho/graphql-kit/service.go",
		"return graphql.MustParseSchema.*",
		patchGraphqlKit,
	)
}

func graphqlGoPermiteUsarCamposNoResolver() bool {
	b, err := ioutil.ReadFile(
		build.Default.GOPATH + "/" + graphqlGoPath + "/graphql.go",
	)
	check(err, "erro ao ler arquivo: %v", err)
	return strings.Contains(string(b), "UseFieldResolvers")
}

// Run runs an arbitrary command
func Run(format string, a ...interface{}) string {
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

// Sed replaces strings in specified file
func Sed(file, old, new string) {
	b, err := ioutil.ReadFile(file)
	check(err, "erro ao ler arquivo '%v': %v", file, err)
	re := regexp.MustCompile(old)
	var str string
	if len(new) > 0 && (new[0] == '&' || new[len(new)-1] == '&') {
		str = re.ReplaceAllStringFunc(string(b), func(s string) string {
			if new[0] == '&' {
				return s + new[1:]
			}
			return new[:len(new)-1] + s
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
	astApply(path, func(c *astutil.Cursor) bool {
		n, ok := c.Node().(*ast.FuncDecl)
		if !ok {
			return true
		}
		if n.Name.Name == fn {
			c.Delete()
			return false
		}
		return true
	})
}

func replaceGprcEncodersAndDecoders(path string) {
	decodeRequest := regexp.MustCompile(`decode(.+)Request`)
	decodeResponse := regexp.MustCompile(`decode(.+)Response`)
	encodeRequest := regexp.MustCompile(`encode(.+)Request`)
	encodeResponse := regexp.MustCompile(`encode(.+)Response`)
	method := func(n *ast.FuncDecl, r *regexp.Regexp) string {
		return r.FindStringSubmatch(n.Name.Name)[1]
	}
	callExpr := func(fn, pkg, reqRes string, n *ast.FuncDecl, r *regexp.Regexp) bool {
		ret, ok := n.Body.List[0].(*ast.ReturnStmt)
		if !ok {
			return false
		}
		id, ok := ret.Results[0].(*ast.Ident)
		if !ok || id.String() != "nil" {
			return false
		}
		ret.Results[0] = &ast.CallExpr{
			Fun: &ast.Ident{Name: fn},
			Args: []ast.Expr{
				&ast.UnaryExpr{
					Op: token.AND,
					X: &ast.CompositeLit{
						Type: &ast.Ident{
							Name: pkg + "." + method(n, r) + reqRes,
						},
					},
				},
				&ast.Ident{Name: n.Type.Params.List[1].Names[0].Name},
			},
		}
		if r == encodeResponse {
			ret.Results[1] =
				&ast.Ident{Name: "r.(endpoint." + method(n, r) + "Response).Err"}
		} else {
			ret.Results[1] = &ast.Ident{Name: "nil"}
		}
		return false
	}
	astApply(path, func(c *astutil.Cursor) bool {
		n, ok := c.Node().(*ast.FuncDecl)
		if !ok {
			return true
		}
		if decodeRequest.MatchString(n.Name.Name) {
			return callExpr("copier.CopyAndDereference", "endpoint", "Request", n, decodeRequest)
		} else if decodeResponse.MatchString(n.Name.Name) {
			return callExpr("copier.CopyAndDereference", "endpoint1", "Response", n, decodeResponse)
		} else if encodeRequest.MatchString(n.Name.Name) {
			return callExpr("copier.Copy", "pb", "Request", n, encodeRequest)
		} else if encodeResponse.MatchString(n.Name.Name) {
			return callExpr("copier.Copy", "pb", "Reply", n, encodeResponse)
		}
		return true
	})
}

func astApply(path string, af func(c *astutil.Cursor) bool) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	check(err, "erro ao fazer parse de '%v': %v", path, err)
	result := astutil.Apply(file, af, nil)
	var buf bytes.Buffer
	err = printer.Fprint(&buf, fset, result)
	check(err, "erro ao imprimir '%v': %v", path, err)
	err = ioutil.WriteFile(path, buf.Bytes(), os.ModePerm)
	check(err, "erro ao gravar '%v': %v", path, err)
	Run("goimports -w " + path)
}

func check(err error, msg string, a ...interface{}) {
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf(msg, a...))
		os.Exit(1)
	}
}
