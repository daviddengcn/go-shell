package gsvm

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"log"
	"math"
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
	Find(ident string) (v reflect.Value, isConst bool)
	// Returns a reflect.Value with pointer to a local variable
	FindLocal(ident string) (v reflect.Value, isConst bool)
	// Adding a reflect.Value with pointer to a new variable
	AddLocal(ident string, v reflect.Value, isConst bool)
	// Returns a namespace for a new block
	NewBlock() NameSpace
}

type theNameSpace struct {
	UpperVars   map[string]reflect.Value
	UpperConsts map[string]reflect.Value
	LocalVars   map[string]reflect.Value
	LocalConsts map[string]reflect.Value
}

func newNameSpace() NameSpace {
	return &theNameSpace{
		UpperVars:   make(map[string]reflect.Value),
		UpperConsts: make(map[string]reflect.Value),
		LocalVars:   make(map[string]reflect.Value),
		LocalConsts: make(map[string]reflect.Value),
	}
}

func (ns *theNameSpace) Find(ident string) (v reflect.Value, isConst bool) {
	if v, isConst := ns.FindLocal(ident); v != noValue {
		return v, isConst
	}
	if v, ok := ns.UpperVars[ident]; ok {
		return v, false
	}
	if v, ok := ns.UpperConsts[ident]; ok {
		return v, true
	}
	return noValue, false
}

func (ns *theNameSpace) FindLocal(ident string) (v reflect.Value, isConst bool) {
	if v, ok := ns.LocalVars[ident]; ok {
		return v, false
	}
	if v, ok := ns.LocalConsts[ident]; ok {
		return v, true
	}
	return noValue, false
}

func (ns *theNameSpace) AddLocal(ident string, v reflect.Value, isConst bool) {
	if isConst {
		ns.LocalConsts[ident] = v
	}
	ns.LocalVars[ident] = v
}

func (ns *theNameSpace) NewBlock() NameSpace {
	newNs := &theNameSpace{
		UpperVars:   make(map[string]reflect.Value),
		UpperConsts: make(map[string]reflect.Value),
		LocalVars:   make(map[string]reflect.Value),
		LocalConsts: make(map[string]reflect.Value),
	}
	// Merge upper vars and local vars as upper vars
	for k, v := range ns.UpperVars {
		newNs.UpperVars[k] = v
	}
	for k, v := range ns.LocalVars {
		newNs.UpperVars[k] = v
	}
	// Merge upper consts and local vars as upper consts
	for k, v := range ns.UpperConsts {
		newNs.UpperConsts[k] = v
	}
	for k, v := range ns.LocalConsts {
		newNs.UpperConsts[k] = v
	}
	return newNs
}

type machine struct {
	GlobalNameSpace NameSpace
	Packages        map[string]map[string]reflect.Value
}

type noValueType interface{}

var (
	noValue = reflect.ValueOf(noValueType(nil))
)

func doSelect(v reflect.Value, sel string) (reflect.Value, error) {
	// TODO
	return noValue, nil
}

func (mch *machine) findSelected(ns NameSpace, x, sel string) (reflect.Value, error) {
	if pv, _ := ns.Find(x); pv != noValue {
		// x is a variable's name
		return doSelect(pv.Elem(), sel)
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
			"math": map[string]reflect.Value{
				"Sin": reflect.ValueOf(math.Sin),
			},
		},
	}
}
