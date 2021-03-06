package pkg

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/golangplus/bytes"

	"github.com/daviddengcn/go-villa"
	gosl "github.com/daviddengcn/gosl/builtin"
	"github.com/golangplus/fmt"
)

var (
	GoRoot  villa.Path
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
	Path  string
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
		fmt.Fprintln(out, `    `+ia.Alias+` `+strconv.Quote(ia.Path))
	}
	fmt.Fprintln(out, `)`)

	fmt.Fprintln(out, `
var(
	valueOf = reflect.ValueOf
	typeOf = gsvm.PtrToTypeValue
)

func elemOf(vl interface{}) reflect.Value {
	return reflect.ValueOf(vl).Elem()
}
`)

	fmtp.Fprintfln(out, "var gImportedPkgs = gsvm.PackageNameSpace{Packages: map[string]gsvm.Package{")
	pkgSrcs := make(map[string]*bytesp.Slice)
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
				pkgSrcs[pkgName] = bytesp.NewPSlice(nil)
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
					fmtp.Fprintfln(pkgSrcs[pkgName], "    %s: valueOf(%s),", strconv.Quote(objName), refName)
				case ast.Typ:
					fmtp.Fprintfln(pkgSrcs[pkgName], "    %s: typeOf((*%s)(nil)),",
						strconv.Quote(objName), refName)
				case ast.Var:
					fmtp.Fprintfln(pkgSrcs[pkgName], "    %s: elemOf(&%s),", strconv.Quote(objName), refName)
				case ast.Fun:
					fmtp.Fprintfln(pkgSrcs[pkgName], "    %s: valueOf(%s),", strconv.Quote(objName), refName)
				}
			}
		}
	}

	for pkgName, src := range pkgSrcs {
		fmtp.Fprintfln(out, "    %s: gsvm.Package{", strconv.Quote(pkgName))
		fmt.Fprintf(out, string(*src))
		fmt.Fprintln(out, "    },")
	}
	fmt.Fprintln(out, "}}")

	return nil
}
