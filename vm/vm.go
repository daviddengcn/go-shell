package gsvm

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"log"
	"reflect"
	"strings"
)

var (
	FragmentErr = errors.New("Fragment")
)

type Machine interface {
	Run(line string) error
}

type NameSpace interface {
	// Returns a reflect.Value
	Find(ident string) (v reflect.Value)
	// Returns a reflect.Value
	FindLocal(ident string) (v reflect.Value)
	// Adding a reflect.Value
	AddLocal(ident string, v reflect.Value)
	// Returns a namespace for a new block
	NewBlock() NameSpace
}

type theNameSpace struct {
	Upper     NameSpace
	LocalVars map[string]reflect.Value
}

func NewNameSpace() NameSpace {
	return &theNameSpace{
		Upper:     nil,
		LocalVars: make(map[string]reflect.Value),
	}
}

func (ns *theNameSpace) Find(ident string) reflect.Value {
	if v := ns.FindLocal(ident); v != NoValue {
		return v
	}

	if ns.Upper != nil {
		return ns.Upper.Find(ident)
	}

	return NoValue
}

func (ns *theNameSpace) FindLocal(ident string) reflect.Value {
	if v, ok := ns.LocalVars[ident]; ok {
		return v
	}
	return NoValue
}

func (ns *theNameSpace) AddLocal(ident string, v reflect.Value) {
	ns.LocalVars[ident] = v
}

func (ns *theNameSpace) NewBlock() NameSpace {
	return NewNameSpaceBlock(ns)
}

func NewNameSpaceBlock(ns NameSpace) NameSpace {
	return &theNameSpace{
		Upper:     ns,
		LocalVars: make(map[string]reflect.Value),
	}
}

type machine struct {
	GlobalNameSpace NameSpace
}

type noValueType interface{}

var (
	NoValue = reflect.ValueOf(noValueType(nil))
)

var (
	trueValue  = reflect.ValueOf(true)
	falseValue = reflect.ValueOf(false)
)

func keywordValue(ident string) reflect.Value {
	switch ident {
	case "true":
		return trueValue
	case "false":
		return falseValue
	default:
		return NoValue
	}
}

func isFragmentError(errList scanner.ErrorList, lastLine int) bool {
	return len(errList) == 1 && errList[0].Pos.Line >= lastLine
}

const (
	srcPrefix = `package main; func main() {
`
	srcSuffix = `
}`
)

func (mch *machine) Run(line string) error {
	src := srcPrefix + line + srcSuffix

	nLines := len(strings.Split(src, "\n"))

	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, "", src, 0)
	if err != nil {
		if isFragmentError(err.(scanner.ErrorList), nLines) {
			return FragmentErr
		}
		ast.Print(token.NewFileSet(), err)
		log.Printf("Syntax error: %v %d", err, nLines)
		return err
	}
	//	log.Println(line)
	for _, st := range f.Decls[0].(*ast.FuncDecl).Body.List {
		if err := mch.runStatement(mch.GlobalNameSpace, st); err != nil {
			return err
		}
	}
	//	log.Println(mch.GlobalNameSpace)
	return nil
}

type Package map[string]reflect.Value

var PackageType = reflect.TypeOf(Package(nil))

func New(initNS NameSpace) Machine {
	return &machine{
		GlobalNameSpace: initNS.NewBlock(),
	}
}

type PackageNameSpace struct {
	Packages map[string]Package
}

func (p *PackageNameSpace) Find(ident string) (v reflect.Value) {
	return p.FindLocal(ident)
}
func (p *PackageNameSpace) FindLocal(ident string) (v reflect.Value) {
	if pkg, ok := p.Packages[ident]; ok {
		return reflect.ValueOf(pkg)
	}

	if pkg, ok := p.Packages[""]; ok {
		if v, ok := pkg[ident]; ok {
			return v
		}
	}

	return NoValue
}
func (p *PackageNameSpace) AddLocal(ident string, v reflect.Value) {
	panic("not implemented")
}
func (p *PackageNameSpace) NewBlock() NameSpace {
	return NewNameSpaceBlock(p)
}
