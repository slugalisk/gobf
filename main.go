package main

import (
	"flag"
	"fmt"
	"path"
	"path/filepath"

	"github.com/slugalisk/gobf/obfuscator"
)

var (
	srcPath    = flag.String("src", "", "path to main package")
	rootPath   = flag.String("root", "", "path to project root (defaults to --src)")
	targetPath = flag.String("target", "", "new GOPATH to copy packages to")
)

func main() {
	flag.Parse()

	options := obfuscator.Options{}

	var err error
	options.SrcPath, err = filepath.Abs(*srcPath)
	if err != nil {
		panic(err)
	}
	options.TargetPath, err = filepath.Abs(*targetPath)
	if err != nil {
		panic(err)
	}

	if rootPath == nil {
		options.RootPath = *srcPath
	} else {
		options.RootPath, err = filepath.Abs(*rootPath)
		if err != nil {
			panic(err)
		}
	}

	alias, err := obfuscator.Rewrite(options)
	if err != nil {
		panic(err)
	}

	fmt.Printf(
		"ready to build:\nGOPATH=%s go build -o %s %s\n",
		*targetPath,
		path.Base(*srcPath),
		alias,
	)
}
