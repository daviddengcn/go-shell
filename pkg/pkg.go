package pkg

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	
	"github.com/daviddengcn/go-villa"
	gosl "github.com/daviddengcn/gosl/builtin"
)

var (
	GoRoot villa.Path
	GoPaths []villa.Path
)

func init() {
	lines := strings.Split(gosl.Eval("go", "env", "GOROOT", "GOPATH"), "\n")
	GoRoot = villa.Path(lines[0])
	paths := strings.Split(lines[1], string(os.PathSeparator))
	GoPaths = make([]villa.Path, 0, len(paths))
	for _, path := range paths {
		GoPaths = append(GoPaths, villa.Path(path))
	}
}

type ImportAs struct {
	Alias string
	Path string
}

func isCaptilized(s string) bool {
	if s == "" {
		return false
	}
	
	return s[0] >= 'A' && s[0] <= 'Z'
}

func GenSource(imports []ImportAs, out io.Writer) error {
	fmt.Fprintln(out, `package main`)
	
	fmt.Fprintln(out, `import(
    "github.com/daviddengcn/go-shell/vm"`)
	for _, ia := range imports {
		fmt.Fprintln(out, `    ` + ia.Alias + ` ` + strconv.Quote(ia.Path))
	}
	fmt.Fprintln(out, `)`)
	
	GoRootSrc := GoRoot.Join("src")
	fmt.Fprintf(out, "var gImportedPkgs = gsvm.PackageNameSpace{Packages: map[string]gsvm.Package{\n")
	for _, ia := range imports {
		if ia.Alias == "_" {
			// side-effect only import
			continue
		}
		
		dir := GoRootSrc.Join(ia.Path)
		fmt.Println("import", ia.Alias, dir)

		//pkgInfo, _ := build.Import(ia.Path, "", build.FindOnly)
		//fmt.Printf("%+v\n", pkgInfo)
		
		fs := token.NewFileSet()
		pkgs, err := parser.ParseDir(fs, dir.S(), func(fi os.FileInfo) bool{
			return !strings.HasSuffix(fi.Name(), "_test.go")
		}, 0)
		if err != nil {
			log.Fatalf("Parsing %s failed: %v", ia.Path, err)
		}
		
		if len(pkgs) > 1 {
			log.Fatalf("More than one packages defined in: %s", ia.Path)
		}
		
		for pkgName, p := range pkgs {
			if ia.Alias == "." {
				pkgName = ""
			} else  if ia.Alias != "" {
				pkgName = ia.Alias
			}
			fmt.Fprintf(out, "  %s: gsvm.Package{\n", strconv.Quote(pkgName))
			
			for _, f := range p.Files {
				for objName, obj := range f.Scope.Objects {
					if !isCaptilized(objName) {
						continue
					}
					
					refName := pkgName
					if pkgName != "" {
						refName = pkgName + "." + objName
					}
					
					switch obj.Kind {
					case ast.Con:
						if objName == "MaxUint64" {
							// a Go bug
							continue
						}
						fmt.Fprintf(out, "    %s: reflect.ValueOf(%s),\n", strconv.Quote(objName), refName)
					case ast.Typ:
						fmt.Fprintf(out, "    %s: reflect.ValueOf(gsvm.TypeValue{reflect.TypeOf((*%s)(nil)).Elem()}),\n",
							strconv.Quote(objName), refName)
					case ast.Var:
						fmt.Fprintf(out, "    %s: reflect.ValueOf(&%s).Elem(),\n", strconv.Quote(objName), refName)
					case ast.Fun:
						fmt.Fprintf(out, "    %s: reflect.ValueOf(%s),\n", strconv.Quote(objName), refName)
					}
				}
			}
			fmt.Fprintf(out, "  },\n")
		}
	}
	fmt.Fprintf(out, "}}\n")

	return nil
}
