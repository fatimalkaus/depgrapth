package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/emicklei/dot"
)

var (
	versionRegex = regexp.MustCompile(`\sv\d+.\d+.\d+`)
	pkgPath      = filepath.Join(build.Default.GOPATH, "pkg/mod")
	maxDepth     = 10
	regex        *regexp.Regexp
)

func main() {
	flag.IntVar(&maxDepth, "depth", 10, "The maximum depth for building the graph")
	var regexStr string
	flag.StringVar(&regexStr, "reg", "", "Specifies the regular expression for filtering the dependencies")
	flag.Parse()
	regex = regexp.MustCompile(regexStr)

	path, _ := os.Getwd()
	if args := flag.Args(); len(args) != 0 {
		path = args[0]
	}

	g := dot.NewGraph(dot.Directed)
	node := g.Node(filepath.Base(path))

	f, err := os.Open(filepath.Join(path, "go.mod"))
	if err != nil {
		panic(err)
	}
	defer f.Close()

	buildGraph(g, node, f, 0)

	os.Stdout.WriteString(g.String())
}

func buildGraph(g *dot.Graph, parent dot.Node, f *os.File, depth int) {
	if depth >= maxDepth {
		return
	}
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "require"))
		if strings.Contains(line, "// indirect") ||
			!versionRegex.MatchString(line) ||
			!regex.MatchString(line) {
			continue
		}

		parts := strings.Fields(line)
		pkg := parts[0]
		node := g.Node(pkg)
		if edges := g.FindEdges(parent, node); len(edges) == 0 {
			g.Edge(parent, node)
		}
		version := parts[1]
		target := fmt.Sprintf("%s@%s", pkg, version)

		gomodPath := filepath.Join(pkgPath, target, "go.mod")
		func() {
			if f, err := os.Open(gomodPath); err == nil {
				defer f.Close()
				buildGraph(g, node, f, depth+1)
			}
		}()
	}
}
