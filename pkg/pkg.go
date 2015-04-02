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
	"go/build"
	
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
	
	fmt.Fprintf(out, "var gImportedPkgs = gsvm.PackageNameSpace{Packages: map[string]gsvm.Package{\n")
	pkgSrcs := make(map[string]*villa.ByteSlice)
	for _, ia := range imports {
		if ia.Alias == "_" {
			// side-effect only import
			continue
		}
		
		fs := token.NewFileSet()
		pkgInfo, _ := build.Import(ia.Path, "", 0)
		dir := villa.Path(pkgInfo.Dir)
		fmt.Println("import", ia.Alias, strconv.Quote(ia.Path))
		for _, goFile := range pkgInfo.GoFiles {
			fn := dir.Join(goFile)
			f, err := parser.ParseFile(fs, fn.S(), nil, 0)
			if err != nil {
				villa.Fatalf("Parse %s failed: %v", fn.S(), err)
			}
			pkgName := f.Name.Name
			if ia.Alias != "" {
				pkgName = ia.Alias
			}
			if _, ok := pkgSrcs[pkgName]; !ok {
				// a new package
				pkgSrcs[pkgName] = villa.NewPByteSlice(nil)
			}
			for objName, obj := range f.Scope.Objects {
				if !isCaptilized(objName) {
					continue
				}
				
				refName := objName
				if pkgName != "" {
					refName = pkgName + "." + objName
				}
				
				switch obj.Kind {
				case ast.Con:
					if objName == "MaxUint64" {
						// a Go bug
						continue
					}
					fmt.Fprintf(pkgSrcs[pkgName], "    %s: reflect.ValueOf(%s),\n", strconv.Quote(objName), refName)
				case ast.Typ:
					fmt.Fprintf(pkgSrcs[pkgName], "    %s: reflect.ValueOf(gsvm.TypeValue{reflect.TypeOf((*%s)(nil)).Elem()}),\n",
						strconv.Quote(objName), refName)
				case ast.Var:
					fmt.Fprintf(pkgSrcs[pkgName], "    %s: reflect.ValueOf(&%s).Elem(),\n", strconv.Quote(objName), refName)
				case ast.Fun:
					fmt.Fprintf(pkgSrcs[pkgName], "    %s: reflect.ValueOf(%s),\n", strconv.Quote(objName), refName)
				}
			}
		}
	}
	
	for pkgName, src := range pkgSrcs {
		fmt.Fprintf(out, "    %s: gsvm.Package{\n", strconv.Quote(pkgName))
		fmt.Fprintf(out, string(*src))
		fmt.Fprintln(out, "    },")
	}
	fmt.Fprintf(out, "}}\n")

	return nil
}
