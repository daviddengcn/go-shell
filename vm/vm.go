package gsvm

import (
	"errors"
	"fmt"
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
	// Returns a reflect.Value with pointer to a variable
	FindVar(ident string) reflect.Value
	// Returns a reflect.Value with pointer to a local variable
	FindLocalVar(ident string) reflect.Value
	// Adding a reflect.Value with pointer to a new variable
	AddLocalVar(ident string, v reflect.Value)
	// Returns a namespace for a new block
	NewBlock() NameSpace
}

type theNameSpace struct {
	UpperVars map[string]reflect.Value
	LocalVars map[string]reflect.Value
}

func newNameSpace() NameSpace {
	return &theNameSpace{
		UpperVars: make(map[string]reflect.Value),
		LocalVars: make(map[string]reflect.Value),
	}
}

func (ns *theNameSpace) FindVar(ident string) reflect.Value {
	if v := ns.FindLocalVar(ident); v != noValue {
		return v
	}
	if v, ok := ns.UpperVars[ident]; ok {
		return v
	}
	return noValue
}

func (ns *theNameSpace) FindLocalVar(ident string) reflect.Value {
	if v, ok := ns.LocalVars[ident]; ok {
		return v
	}
	return noValue
}

func (ns *theNameSpace) AddLocalVar(ident string, v reflect.Value) {
	ns.LocalVars[ident] = v
}

func (ns *theNameSpace) NewBlock() NameSpace {
	newNs := &theNameSpace{
		UpperVars: make(map[string]reflect.Value),
		LocalVars: make(map[string]reflect.Value),
	}
	for k, v := range ns.UpperVars {
		newNs.UpperVars[k] = v
	}
	for k, v := range ns.LocalVars {
		newNs.UpperVars[k] = v
	}
	return newNs
}

type machine struct {
	GlobalNameSpace NameSpace
	Packages        map[string]map[string]reflect.Value
}

var (
	noValue = reflect.ValueOf(nil)
)

func doSelect(v reflect.Value, sel string) (reflect.Value, error) {
	// TODO
	return noValue, nil
}

func (mch *machine) findSelected(ns NameSpace, x, sel string) (reflect.Value, error) {
	if xv := ns.FindVar(x); xv != noValue {
		// x is a variable's name
		return doSelect(reflect.Indirect(xv), sel)
	}

	if funcs, ok := mch.Packages[x]; ok {
		// x is a package name
		if f, ok := funcs[sel]; ok {
			return f, nil
		}
		return noValue, fmt.Errorf("Undefined: %s.%s", x, sel)
	}

	return noValue, fmt.Errorf("Undefined: %s", x)
}

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
		return noValue
	}
}

func leftCompatible(x reflect.Value, op token.Token) bool {
	return true
}

func matchType(x, y reflect.Value) (nX, nY reflect.Value, err error) {
	return x, y, nil
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

func New() Machine {
	return &machine{
		GlobalNameSpace: newNameSpace(),
		Packages: map[string]map[string]reflect.Value{
			"fmt": map[string]reflect.Value{
				"Println": reflect.ValueOf(fmt.Println),
				"Print":   reflect.ValueOf(fmt.Print),
			},
			"reflect": map[string]reflect.Value{
				"TypeOf": reflect.ValueOf(reflect.TypeOf),
			},
		},
	}
}
