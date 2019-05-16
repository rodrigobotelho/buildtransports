package main

import (
	"fmt"
	"os"
	"path/filepath"

	builder "github.com/rodrigobotelho/buildtransports/pkg"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(
			os.Stderr,
			"Uso: %v [opções] <nome de servico>\n",
			filepath.Base(os.Args[0]),
		)
		os.Exit(1)
	}
	customName := ""
	if len(os.Args) > 2 {
		customName = os.Args[2]
	}
	builder.Build(os.Args[1], customName)
}
