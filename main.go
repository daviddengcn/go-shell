package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"

	"github.com/daviddengcn/go-ljson-conf"
	"github.com/daviddengcn/go-shell/pkg"
	"github.com/daviddengcn/go-villa"
)

func genFilename() villa.Path {
	if true {
		return villa.Path("/tmp")
	}
	dir := villa.Path(os.TempDir())
	for {
		base := villa.Path(fmt.Sprintf("go-shell-%08x", rand.Int63n(math.MaxInt64)))
		fn := dir.Join(base)
		if !fn.Exists() {
			return fn
		}
	}
}

const mainGoSrc = `package main

import(
	"github.com/daviddengcn/go-shell/shell"
)

func main() {
	shell.Run(&gImportedPkgs)
}
`

func main() {
	conf, _ := ljconf.Load("shell.json")
	imports := conf.Object("import", nil)
	importList := make([]pkg.ImportAs, 0, len(imports))
	for p, a := range imports {
		importList = append(importList, pkg.ImportAs{Alias: fmt.Sprint(a), Path: p})
	}

	base := genFilename()
	fmt.Println("base", base)
	if err := base.MkdirAll(0755); err != nil {
		log.Fatalf("Mkdirs failed: %v", err)
	}
	fnMainGo := base.Join("main.go")
	if err := ioutil.WriteFile(fnMainGo.S(), []byte(mainGoSrc), 0644); err != nil {
		log.Fatalf("WriteFile to %s failed: %v", fnMainGo, err)
	}

	fnPkgGo := base.Join("pkg.go")
	func() {
		f, err := fnPkgGo.Create()
		if err != nil {
			log.Fatalf("Create file %s failed: %v", fnPkgGo, err)
		}
		defer f.Close()

		if err := pkg.GenSource(importList, f); err != nil {
			log.Fatalf("GenSource failed: %v", err)
		}
	}()

	cmd := villa.Path("go").Command("run", fnMainGo.S(), fnPkgGo.S())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		log.Fatalf("go run failed: %v", err)
	}
}
