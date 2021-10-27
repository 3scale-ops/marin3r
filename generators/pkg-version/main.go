package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	. "github.com/dave/jennifer/jen"

	"github.com/spf13/cobra"
)

const (
	sourceModuleName string = "github.com/envoyproxy/go-control-plane"
)

var (
	version     string
	packageName string
	packageFile string
)

var cmd = &cobra.Command{
	Use: "gen-pkg-version",
	Run: func(cmd *cobra.Command, args []string) {
		generate()
	},
}

func init() {
	// flags
	cmd.Flags().StringVar(&version, "version", "", "The version (only the number, without the leading 'v')")
	cmd.MarkFlagRequired("version")
	cmd.Flags().StringVar(&packageName, "package-name", "version", "The package name")
	cmd.Flags().StringVar(&packageFile, "package-file", "zz_generated.go", "The package file")

}

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate() {
	log.Printf("generating for %s", version)
	f, err := os.OpenFile(packageFile, os.O_CREATE|os.O_RDWR, 0644)
	checkIfError(err)
	defer f.Close()

	pkg := NewFile(packageName)
	pkg.Const().Defs(Id("version").String().Op("=").Lit("v" + version))

	w := bufio.NewWriter(f)
	err = pkg.Render(w)
	checkIfError(err)
	w.Flush()
}

func checkIfError(err error) {
	if err == nil {
		return
	}

	fmt.Printf("\x1b[31;1m%s\x1b[0m\n", fmt.Sprintf("error: %s", err))
	os.Exit(1)
}
