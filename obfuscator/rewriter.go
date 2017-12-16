package obfuscator

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/otiai10/copy"
)

const (
	vendorPath = "/vendor/"
)

// Options ...
type Options struct {
	SrcPath    string
	RootPath   string
	TargetPath string
}

// Rewrite target project
func Rewrite(options Options) (string, error) {
	context := build.Default

	pkg, err := context.ImportDir(options.SrcPath, 0)
	if err != nil {
		return "", err
	}

	r := &rewriter{
		options: options,
		namer:   NewNamer(5),
		seen:    make(map[string]struct{}),
	}
	alias, err := r.RewritePackage(pkg)
	if err != nil {
		return "", err
	}

	return alias, nil
}

type rewriter struct {
	options Options
	namer   *Namer
	seen    map[string]struct{}
}

func (r *rewriter) RewritePackage(pkg *build.Package) (string, error) {
	if _, ok := r.seen[pkg.Dir]; ok {
		return r.namer.Alias(pkg.Dir)
	}
	r.seen[pkg.Dir] = struct{}{}

	if strings.HasPrefix(pkg.Dir, os.Getenv("GOROOT")) {
		r.namer.Assign(pkg.ImportPath, pkg.ImportPath)
		r.namer.Assign(pkg.Dir, pkg.Dir)
		return pkg.Dir, nil
	}

	names := []string{pkg.ImportPath, pkg.Dir}
	vendorPathIndex := strings.LastIndex(pkg.Dir, vendorPath)
	if vendorPathIndex != -1 {
		names = append(names, pkg.Dir[vendorPathIndex+len(vendorPath):])
	}

	alias, err := r.namer.AliasAll(names)
	if err != nil {
		return "", err
	}
	copy.Copy(pkg.Dir, path.Join(r.options.TargetPath, "src", alias))

	var paths []string
	paths = append(paths, pkg.GoFiles...)
	prefixDirectory(pkg.Dir, paths)

	for _, path := range paths {
		err := r.rewriteFile(pkg, path)
		if err != nil {
			return "", err
		}
	}

	return alias, nil
}

func (r *rewriter) rewriteFile(pkg *build.Package, src string) error {
	code, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, src, code, parser.AllErrors)
	if err != nil {
		return err
	}

	dirAlias, err := r.namer.Alias(pkg.Dir)
	if err != nil {
		return err
	}
	srcAlias, err := r.namer.Alias(src)
	if err != nil {
		return err
	}
	srcAlias = fmt.Sprintf("%s.go", srcAlias)

	for _, imp := range file.Imports {
		err := r.rewriteImport(filepath.Dir(src), imp)
		if err != nil {
			return err
		}
	}

	oldPath := path.Join(r.options.TargetPath, "src", dirAlias, path.Base(src))
	err = os.Remove(oldPath)
	if err != nil {
		return err
	}

	newPath := path.Join(r.options.TargetPath, "src", dirAlias, srcAlias)
	fw, err := os.OpenFile(newPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	if err := format.Node(fw, fset, file); err != nil {
		return err
	}

	return nil
}

func (r *rewriter) rewriteImport(srcDir string, imp *ast.ImportSpec) error {
	importPath := imp.Path.Value[1 : len(imp.Path.Value)-1]
	pkg, err := build.Default.Import(importPath, srcDir, 0)
	if err != nil {
		return err
	}
	r.RewritePackage(pkg)

	alias, err := r.namer.Alias(importPath)
	if err != nil {
		return err
	}

	imp.Path.Value = fmt.Sprintf(`"%s"`, alias)

	return nil
}

func prefixDirectory(directory string, names []string) {
	if directory != "." {
		for i, name := range names {
			names[i] = filepath.Join(directory, name)
		}
	}
}
