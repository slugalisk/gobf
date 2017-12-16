package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/fatih/astrewrite"
	"github.com/otiai10/copy"
)

var testFile = "/home/user/go/src/github.com/slugalisk/gobf/test/saas/sites/cmd/rest-server/main.go"

// Obfuscator ...
type Obfuscator struct {
	fset      *token.FileSet
	symbols   map[string]string
	filenames map[string]string
}

// we may need to know which compiler we're going to use for package
// resolution to work
// https://golang.org/pkg/go/importer/#For

// --project-root
// --bin-src-path

// NewObfuscator ...
func NewObfuscator() *Obfuscator {
	return &Obfuscator{}
}

// Memes parse a file
// this gets us an ast for a asingle file or an ast.Package with parsed files
// in it... this is shallow at best and doesn't expose file names
func (o *Obfuscator) Memes(filename string) error {
	fset := token.NewFileSet()
	// symbols := make(map[string]string)
	// filenames := make(map[string]string)

	src, err := ioutil.ReadFile(testFile)
	if err != nil {
		return err
	}

	// Parse src but stop after processing the imports.
	f, err := parser.ParseFile(fset, filename, src, parser.AllErrors)
	if err != nil {
		return err
	}

	spew.Dump(f)

	// Print the imports from the file's AST.
	for _, s := range f.Imports {
		log.Println(s.Path.Value, build.IsLocalImport(s.Path.Value))

		spew.Dump(s)
	}

	// -----------------------------------------------------------------------

	// // Create an ast.CommentMap from the ast.File's comments.
	// // This helps keeping the association between comments
	// // and AST nodes.
	// cmap := ast.NewCommentMap(fset, f, f.Comments)

	// // Remove the first variable declaration from the list of declarations.
	// for i, decl := range f.Decls {
	// 	if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.VAR {
	// 		copy(f.Decls[i:], f.Decls[i+1:])
	// 		f.Decls = f.Decls[:len(f.Decls)-1]
	// 	}
	// }

	// // Use the comment map to filter comments that don't belong anymore
	// // (the comments associated with the variable declaration), and create
	// // the new comments list.
	// f.Comments = cmap.Filter(f).Comments()

	// Print the modified AST.
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, f); err != nil {
		panic(err)
	}
	fmt.Println("---------------------------------------------")
	fmt.Printf("%s", buf.Bytes())
	fmt.Println("---------------------------------------------")

	// Output:
	// // This is the package comment.
	// package main
	//
	// // This comment is associated with the hello constant.
	// const hello = "Hello, World!" // line comment 1
	//
	// // This comment is associated with the main function.
	// func main() {
	// 	fmt.Println(hello) // line comment 3
	// }

	return nil
}

// Memes2 start from an importer
// this seems to get us the entire dependency tree but i'm not sure how
// to get the file paths from the types.Packages... the path only exists
// in .comment which is only included for debugging...
func (o *Obfuscator) Memes2(dirname string) error {
	// fset := token.NewFileSet()
	// fs, err := parser.ParseDir(fset, dirname, nil, parser.AllErrors)
	// if err != nil {
	// 	return err
	// }

	imp := importer.For("source", nil).(types.ImporterFrom)
	pkg, err := imp.ImportFrom(
		"github.com/slugalisk/gobf/test/saas/sites/cmd/rest-server",
		"/home/user/go/src/github.com/slugalisk/gobf/test/saas",
		0,
	)
	if err != nil {
		return err
	}

	t := NewTraverser()
	t.traverseImports(pkg, 0)

	// log.Println(importer.Lookup("github.com/slugalisk/gobf/test/saas/sites/restapi"))

	// for _, f := range fs {
	// 	// mp := importer.For("source", imp)
	// 	for _, s := range f.Files {
	// 		s.Scope.Lookup()
	// 		for _, i := range s.Imports {
	// 			spew.Dump(imp.Import(i.Path.Value[1 : len(i.Path.Value)-2]))
	// 		}
	// 	}
	// }

	return nil
}

// Memes3 start from a build context...
func (o *Obfuscator) Memes3(dir string) error {
	context := build.Default

	pkg, err := context.ImportDir(dir, 0)
	if err != nil {
		return err
	}

	// var names []string
	// names = append(names, pkg.GoFiles...)
	// // names = append(names, pkg.CgoFiles...)
	// // names = append(names, pkg.TestGoFiles...)
	// // names = append(names, pkg.SFiles...)

	// prefixDirectory(dir, names)
	// log.Println(names)

	// ------------------------

	t := NewTraverser2()
	err = t.Load(pkg)
	if err != nil {
		return err
	}

	// for _, name := range names {
	// 	t.readPath(name)

	// 	// imports, err := readPath(name)
	// 	// if err != nil {
	// 	// 	return err
	// 	// }

	// 	// for _, i := range imports {
	// 	// 	p, err := context.Import(
	// 	// 		i.Path.Value[1:len(i.Path.Value)-1],
	// 	// 		name,
	// 	// 		0,
	// 	// 	)
	// 	// 	if err != nil {
	// 	// 		return err
	// 	// 	}
	// 	// 	var ns []string
	// 	// 	ns = append(ns, pkg.GoFiles...)

	// 	// 	prefixDirectory(p.Dir, ns)

	// 	// 	log.Println(ns)
	// 	// }
	// }

	// ------------------------

	return nil
}

// Traverser2 ...
type Traverser2 struct {
	seen  map[string]struct{}
	names map[string]string
	used  map[string]struct{}
}

// NewTraverser2 ...
func NewTraverser2() *Traverser2 {
	return &Traverser2{
		seen:  make(map[string]struct{}),
		names: make(map[string]string),
		used:  make(map[string]struct{}),
	}
}

// RenameAll assign the same alias to all the names given
// if we've already aliased one of the names use the existing alias
func (t *Traverser2) RenameAll(names []string, capitalize bool) (string, error) {
	for _, name := range names {
		if replacement, ok := t.names[name]; ok {
			for _, name := range names {
				t.names[name] = replacement
			}
			return replacement, nil
		}
	}

	unsafe := regexp.MustCompile("[^a-zA-Z]")

	for {
		data := make([]byte, 16)
		if _, err := rand.Read(data); err != nil {
			return "", errors.New("ran out of entropy")
		}
		replacement := base64.URLEncoding.EncodeToString(data)
		replacement = unsafe.ReplaceAllString(replacement, "")

		if capitalize {
			replacement = strings.ToUpper(replacement[:1]) + replacement[1:]
		} else {
			replacement = strings.ToLower(replacement[:1]) + replacement[1:]
		}

		replacement = replacement[0:5]

		if _, ok := t.used[replacement]; !ok {
			t.used[replacement] = struct{}{}
			for _, name := range names {
				t.names[name] = replacement
			}
			return replacement, nil
		}
	}
}

// Rename ...
func (t *Traverser2) Rename(name string, capitalize bool) (string, error) {
	if _, ok := t.used[name]; ok {
		return name, nil
	}

	return t.RenameAll([]string{name}, capitalize)
}

// Load ...
func (t *Traverser2) Load(pkg *build.Package) error {
	if _, ok := t.seen[pkg.Dir]; ok {
		return nil
	}
	t.seen[pkg.Dir] = struct{}{}

	// copy dir ...
	if strings.HasPrefix(pkg.Dir, os.Getenv("GOROOT")) {
		t.names[pkg.ImportPath] = pkg.ImportPath
		t.names[pkg.Dir] = pkg.Dir
		return nil
	}

	vendored := strings.TrimPrefix(pkg.Dir, "/home/user/go/src/github.com/slugalisk/gobf/test/saas/vendor/")

	replacement, err := t.RenameAll([]string{pkg.ImportPath, pkg.Dir, vendored}, false)
	if err != nil {
		return err
	}
	log.Printf("%s, %s => %s", pkg.Dir, pkg.ImportPath, fmt.Sprintf("/tmp/scratch/src/%s", replacement))
	copy.Copy(pkg.Dir, fmt.Sprintf("/tmp/scratch/src/%s", replacement))

	// log.Println(pkg.Dir)
	// spew.Dump(names)

	var names []string
	names = append(names, pkg.GoFiles...)
	prefixDirectory(pkg.Dir, names)

	for _, name := range names {
		err := t.readPath(pkg, name)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *Traverser2) readPath(pkg *build.Package, file string) error {

	context := build.Default

	fset := token.NewFileSet()

	src, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	// ---------------

	// generate the aliases first for recursive dependencies...
	dn, _ := t.Rename(pkg.Dir, false)
	fn, _ := t.Rename(file, false)

	// ---------------

	f, err := parser.ParseFile(fset, file, src, parser.AllErrors)
	if err != nil {
		return err
	}

	for _, i := range f.Imports {
		trimmed := i.Path.Value[1 : len(i.Path.Value)-1]
		p, err := context.Import(
			trimmed,
			filepath.Dir(file),
			0,
		)
		if err != nil {
			return err
		}
		t.Load(p)

		replacement, err := t.Rename(trimmed, false)
		// name := ""
		// if i.Name != nil {
		// 	name = i.Name.Name
		// }

		i.Comment = &ast.CommentGroup{
			List: []*ast.Comment{
				&ast.Comment{
					Slash: i.End(),
					Text:  fmt.Sprintf("// %s", i.Path.Value),
				},
			},
		}
		// log.Printf("%s (%s) => %s", name, i.Path.Value, fmt.Sprintf(`"%s"`, replacement))
		i.Path.Value = fmt.Sprintf(`"%s"`, replacement)
	}

	// ---------------

	err = os.Remove(fmt.Sprintf("/tmp/scratch/src/%s/%s", dn, path.Base(file)))
	if err != nil {
		log.Printf("maybe duplicating shit %s %s /tmp/scratch/src/%s", pkg.Dir, fmt.Sprintf("/tmp/scratch/src/%s/%s.go", dn, path.Base(file)), dn)
		// return err
	}

	rwfn := func(n ast.Node) (ast.Node, bool) {
		switch n.(type) {
		case *ast.TypeSpec:
			x, _ := n.(*ast.TypeSpec)
			x.Name.Name, _ = t.Rename(x.Name.Name, true)
			return x, true
		case *ast.FuncDecl:
			x, _ := n.(*ast.FuncDecl)
			x.Name.Name, _ = t.Rename(x.Name.Name, isCapitalized(x.Name.Name))

			// for _, p := range x.Type.Params.List {
			// 	// log.Println(p.Type.)
			// 	// log.Println(p.Names[0].Name)
			// }

			// x.Name.Name, _ = t.Rename(x.Name.Name, true)
			return x, true
		case *ast.FuncLit:
			x, _ := n.(*ast.FuncLit)

			// for _, p := range x.Type.Params.List {
			// 	switch p.Type.(type) {
			// 	case *ast.BadExpr:
			// 		y, _ := p.Type.(*ast.BadExpr)
			// 		log.Println("*ast.BadExpr", y)
			// 	case *ast.Ident:
			// 		y, _ := p.Type.(*ast.Ident)
			// 		log.Println("*ast.Ident", y)
			// 	case *ast.Ellipsis:
			// 		y, _ := p.Type.(*ast.Ellipsis)
			// 		log.Println("*ast.Ellipsis", y)
			// 	case *ast.BasicLit:
			// 		y, _ := p.Type.(*ast.BasicLit)
			// 		log.Println("*ast.BasicLit", y)
			// 	case *ast.FuncLit:
			// 		y, _ := p.Type.(*ast.FuncLit)
			// 		log.Println("*ast.FuncLit", y)
			// 	case *ast.CompositeLit:
			// 		y, _ := p.Type.(*ast.CompositeLit)
			// 		log.Println("*ast.CompositeLit", y)
			// 	case *ast.ParenExpr:
			// 		y, _ := p.Type.(*ast.ParenExpr)
			// 		log.Println("*ast.ParenExpr", y)
			// 	case *ast.SelectorExpr:
			// 		y, _ := p.Type.(*ast.SelectorExpr)
			// 		pkg, ok := y.X.(*ast.Ident)
			// 		if ok {
			// 			log.Println(pkg.Name, ".", y.Sel.Name)
			// 		}
			// 		if y.Sel.Obj == nil {
			// 			// y.Sel.Name, _ = t.Rename(y.Sel.Name, isCapitalized(y.Sel.Name))
			// 		} // or if it was imported from a non-core package...

			// 		log.Println("*ast.SelectorExpr", y, "--", y.Sel.Name)
			// 	case *ast.IndexExpr:
			// 		y, _ := p.Type.(*ast.IndexExpr)
			// 		log.Println("*ast.IndexExpr", y)
			// 	case *ast.SliceExpr:
			// 		y, _ := p.Type.(*ast.SliceExpr)
			// 		log.Println("*ast.SliceExpr", y)
			// 	case *ast.TypeAssertExpr:
			// 		y, _ := p.Type.(*ast.TypeAssertExpr)
			// 		log.Println("*ast.TypeAssertExpr", y)
			// 	case *ast.CallExpr:
			// 		y, _ := p.Type.(*ast.CallExpr)
			// 		log.Println("*ast.CallExpr", y)
			// 	case *ast.StarExpr:
			// 		y, _ := p.Type.(*ast.StarExpr)
			// 		log.Println("*ast.StarExpr", y, reflect.TypeOf(y.X))
			// 	case *ast.UnaryExpr:
			// 		y, _ := p.Type.(*ast.UnaryExpr)
			// 		log.Println("*ast.UnaryExpr", y)
			// 	case *ast.BinaryExpr:
			// 		y, _ := p.Type.(*ast.BinaryExpr)
			// 		log.Println("*ast.BinaryExpr", y)
			// 	case *ast.KeyValueExpr:
			// 		y, _ := p.Type.(*ast.KeyValueExpr)
			// 		log.Println("*ast.KeyValueExpr", y)
			// 	case *ast.ArrayType:
			// 		y, _ := p.Type.(*ast.ArrayType)
			// 		log.Println("*ast.ArrayType", y)
			// 	case *ast.StructType:
			// 		y, _ := p.Type.(*ast.StructType)
			// 		log.Println("*ast.StructType", y)
			// 	case *ast.FuncType:
			// 		y, _ := p.Type.(*ast.FuncType)
			// 		log.Println("*ast.FuncType", y)
			// 	case *ast.InterfaceType:
			// 		y, _ := p.Type.(*ast.InterfaceType)
			// 		log.Println("*ast.InterfaceType", y)
			// 	case *ast.MapType:
			// 		y, _ := p.Type.(*ast.MapType)
			// 		log.Println("*ast.MapType", y)
			// 	case *ast.ChanType:
			// 		y, _ := p.Type.(*ast.ChanType)
			// 		log.Println("*ast.ChanType", y)
			// 	}
			// }

			return x, true
		case *ast.FuncType:
			x, _ := n.(*ast.FuncType)
			// log.Println(x.Func)
			return x, true
		case *ast.Field:
			x, _ := n.(*ast.Field)
			spew.Dump(x)

			b := bytes.Buffer{}
			format.Node(&b, fset, x)
			log.Println(">>>", b.String())

			return x, true
		case *ast.Ident:
			x, _ := n.(*ast.Ident)
			if x.Obj != nil {
				// log.Printf("%s.%s", x.Obj.Name, x.Name)
				x.Obj.Name, _ = t.Rename(x.Obj.Name, isCapitalized(x.Obj.Name))
			} else {
				// log.Println(x.Name)
			}
			x.Name, _ = t.Rename(x.Name, isCapitalized(x.Name))
			return x, true
		default:
			return n, true
		}
	}
	astrewrite.Walk(f, rwfn)

	fw, err := os.OpenFile(fmt.Sprintf("/tmp/scratch/src/%s/%s.go", dn, fn), os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	if err := format.Node(fw, fset, f); err != nil {
		return err
	}

	// ---------------

	// new path...
	// dir := filepath.Dir(file)
	// sub, ok := paths[dir]

	// rewrite ast

	return nil
}

func isCapitalized(str string) bool {
	return strings.ToUpper(str[:1]) == str[:1]
}

// prefixDirectory places the directory name on the beginning of each name in the list.
func prefixDirectory(directory string, names []string) {
	if directory != "." {
		for i, name := range names {
			names[i] = filepath.Join(directory, name)
		}
	}
}

// Traverser ...
type Traverser struct {
	seen map[string]struct{}
	fset *token.FileSet
	// symbols := make(map[string]string)
	// filenames := make(map[string]string)
}

// NewTraverser ...
func NewTraverser() *Traverser {
	return &Traverser{
		seen: make(map[string]struct{}),
		fset: token.NewFileSet(),
	}
}

func (t *Traverser) traverseImports(pkg *types.Package, depth int) {
	if _, ok := t.seen[pkg.Path()]; ok {
		return
	}
	t.seen[pkg.Path()] = struct{}{}

	prefix := strings.Repeat("-", depth)
	log.Println(prefix, pkg.Name(), pkg.Path())

	scope := pkg.Scope()
	for i := 0; i < scope.NumChildren(); i++ {
		spew.Dump(scope.Child(i).String)
	}

	// err := t.ReadDir(pkg.Path())
	// if err != nil {
	// 	log.Println(err)
	// }

	for _, p := range pkg.Imports() {
		t.traverseImports(p, depth+1)
	}
}

// ReadDir ...
func (t *Traverser) ReadDir(dir string) error {
	pkgs, err := parser.ParseDir(t.fset, dir, nil, parser.AllErrors)
	if err != nil {
		return err
	}

	for _, p := range pkgs {
		for _, f := range p.Files {
			log.Println(f.Name.String())
		}
	}

	return nil
}

func main() {
	obfuscator := NewObfuscator()

	// err := obfuscator.Memes(testFile)
	err := obfuscator.Memes3("/home/user/go/src/github.com/slugalisk/gobf/test/saas/sites/cmd/rest-server")
	if err != nil {
		panic(err)
	}
}
