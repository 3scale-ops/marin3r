package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	. "github.com/dave/jennifer/jen"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/spf13/cobra"
	"golang.org/x/mod/modfile"
)

const (
	sourceModuleName string = "github.com/envoyproxy/go-control-plane"
)

var (
	apiVersion  string
	gomodFile   string
	packagFile  string
	packageName string
)

var cmd = &cobra.Command{
	Use: "gen-pkg-envoy-proto",
	Run: func(cmd *cobra.Command, args []string) {
		generate()
	},
}

func init() {
	// flags
	cmd.Flags().StringVar(&apiVersion, "api-version", "", "Generate imports for proto messages for the specific API version")
	cmd.MarkFlagRequired("api-version")
	cmd.Flags().StringVar(&packagFile, "package-file", "", "Name of the generated package file")
	cmd.MarkFlagRequired("package-file")
	cmd.Flags().StringVar(&gomodFile, "gomod-file", "go.mod", "Location of the go.mod file to extrac go-control-plane release from")
	cmd.Flags().StringVar(&packageName, "package-name", "envoy", "Package name of the generated file")
}

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate() {
	version, isPseudoVersion := inspectVersion()

	log.Printf("generating for %s@%s", sourceModuleName, version)

	list := listProtoPackages(version, isPseudoVersion)

	writePackageFile(packagFile, list)

}

func inspectVersion() (string, bool) {
	body, err := os.ReadFile(gomodFile)
	checkIfError(err)

	gomod, err := modfile.Parse(gomodFile, body, nil)
	checkIfError(err)

	var present bool
	var isPseudoVersion bool
	var ref string

	for _, m := range gomod.Require {
		if m.Mod.Path == sourceModuleName {
			ref = m.Mod.Version
			present = true
		}
	}

	if !present {
		log.Fatalf(fmt.Sprintf("%s module not found in modules file", sourceModuleName))
		return "", false
	}

	exp := regexp.MustCompile(`.*-([0-9a-f]{12})$`)
	if match := exp.FindStringSubmatch(ref); len(match) != 0 {
		// it's a pseudoversion
		ref = match[1]
		isPseudoVersion = true
	}

	return ref, isPseudoVersion

}

func listProtoPackages(ref string, isPseudoVersion bool) []string {
	var list []string

	// var refName plumbing.ReferenceName
	// if isPseudoVersion {
	// 	refName = plumbing.NewBranchReferenceName("main")
	// 	// ref = plumbing.NewHashReference(n plumbing.ReferenceName, h plumbing.Hash)
	// } else {
	// 	refName = plumbing.NewTagReferenceName(ref)
	// }
	// log.Printf("ReferenceName: %s", refName.String())
	repo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL:           "https://" + sourceModuleName,
		Depth:         1,
		ReferenceName: plumbing.NewBranchReferenceName("main"),
		SingleBranch:  true,
	})
	checkIfError(err)

	// Get tree for the given tag
	var revision plumbing.Revision
	if isPseudoVersion {
		revision = plumbing.Revision(ref)
	} else {
		revision = plumbing.Revision(plumbing.NewTagReferenceName(ref))
	}
	log.Printf("revision: %s", revision.String())
	h, err := repo.ResolveRevision(revision)
	checkIfError(err)
	commit, err := repo.CommitObject(*h)
	checkIfError(err)
	tree, err := commit.Tree()
	checkIfError(err)

	// Find packages that containg protobuffer message definitions
	index := map[string]int{}
	tree.Files().ForEach(func(f *object.File) error {
		dir, filename := filepath.Split(f.Name)
		dir = filepath.Clean(dir)
		// Look only under path "envoy/"
		if !strings.HasPrefix(dir, "envoy/") && !strings.HasPrefix(dir, "contrib/envoy/") {
			return nil
		}
		// Find all proto files for the specified API version
		if strings.HasSuffix(filename, ".pb.go") && strings.HasSuffix(dir, "/"+apiVersion) {
			index[dir] = 1
		}
		return nil
	})

	for proto := range index {
		list = append(list, sourceModuleName+"/"+proto)
	}
	sort.Strings(list)

	return list
}

func writePackageFile(packagePath string, importList []string) {

	f, err := os.OpenFile(packagePath, os.O_CREATE|os.O_RDWR, 0644)
	checkIfError(err)
	defer f.Close()

	// Reset file contents before writing
	f.Truncate(0)
	f.Seek(0, 0)

	pkg := NewFile(packageName)
	pkg.Anon(importList...)

	w := bufio.NewWriter(f)
	err = pkg.Render(w)
	checkIfError(err)
}

func checkIfError(err error) {
	if err == nil {
		return
	}

	fmt.Printf("\x1b[31;1m%s\x1b[0m\n", fmt.Sprintf("error: %s", err))
	os.Exit(1)
}
