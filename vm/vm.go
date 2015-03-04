package gsvm

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"log"
	"reflect"
	"strconv"
	"strings"
)

type Machine interface {
	Run(line string) (isFragment bool)
}

type NameSpace interface {
	// Returns a reflect.Value with pointer to a variable
	FindVar(ident string) reflect.Value
	// Returns a reflect.Value with pointer to a local variable
	FindLocalVar(ident string) reflect.Value
	// Adding a reflect.Value with pointer to a new variable
	AddLocalVar(ident string, v reflect.Value)
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

func checkSingleValue(vls []reflect.Value, err error) (reflect.Value, error) {
	if err !=  nil {
		return noValue, err
	}
	if len(vls) != 1 {
		return noValue, fmt.Errorf("multiple-value(%d) in single-value context", len(vls))
	}
	return vls[0], nil
}

func fromSingleValue(vl reflect.Value, err error) ([]reflect.Value, error) {
	if err != nil {
		return nil, err
	}
	return []reflect.Value{vl}, nil
}

// Returns slice of values themselves not the pointers.
func (mch *machine) evalExpr(ns NameSpace, expr ast.Expr) ([]reflect.Value, error) {
	switch expr := expr.(type) {
	case *ast.BasicLit:
		switch expr.Kind {
		case token.INT:
			i, _ := strconv.Atoi(expr.Value)
			return []reflect.Value{reflect.ValueOf(i)}, nil

		case token.STRING:
			s, _ := strconv.Unquote(expr.Value)
			return []reflect.Value{reflect.ValueOf(s)}, nil

		default:
			log.Println("Unknown BasicLit Kind")
			ast.Print(token.NewFileSet(), expr)
			return []reflect.Value{reflect.ValueOf(expr)}, nil
		}

	case *ast.Ident:
		v := ns.FindVar(expr.Name)
		if v == noValue {
			return nil, fmt.Errorf("Unknown Ident %v", expr.Name)
		}
		return []reflect.Value{reflect.Indirect(v)}, nil

	case *ast.CallExpr:
		fn, err := checkSingleValue(mch.evalExpr(ns, expr.Fun))
		if err != nil {
			return nil, err
		}

		if fn.Kind() != reflect.Func {
			return nil, fmt.Errorf("cannot call non-function (type %s)", fn.Type())
		}

		// TODO when len(expr.Args) == 1, check multi return value situation
		args := make([]reflect.Value, len(expr.Args))
		for i, arg := range expr.Args {
			argV, err := checkSingleValue(mch.evalExpr(ns, arg))
			if err != nil {
				return nil, err
			}
			args[i] = argV
		}

		return fn.Call(args), nil

	case *ast.SelectorExpr:
		switch x := expr.X.(type) {
		case *ast.Ident:
			return fromSingleValue(mch.findSelected(ns, x.Name, expr.Sel.Name))
		default:
			log.Println("Unknown SelectorExpr X type")
			ast.Print(token.NewFileSet(), x)
			return []reflect.Value{reflect.ValueOf(expr)}, nil
		}

	default:
		log.Println("Unknown expr type")
		ast.Print(token.NewFileSet(), expr)
		return []reflect.Value{reflect.ValueOf(expr)}, nil
	}
}

func (mch *machine) runStatement(ns NameSpace, st ast.Stmt) error {
	switch st := st.(type) {
	case *ast.AssignStmt:
		// TODO when len(st.Rhs) == 1, check multi value return
		if len(st.Lhs) != len(st.Rhs) {
			return fmt.Errorf("assignment count mismatch: %d %s %d", len(st.Lhs), st.Tok, len(st.Rhs))
		}
		if st.Tok == token.DEFINE {
			hasNew := false
			for _, l := range st.Lhs {
				lIdent := l.(*ast.Ident)
				if v := ns.FindLocalVar(lIdent.Name); v == noValue {
					hasNew = true
				}
			}
			if !hasNew {
				return fmt.Errorf("no new on left side of :=")
			}

			for i, l := range st.Lhs {
				lIdent := l.(*ast.Ident)
				r := st.Rhs[i]
				if v := ns.FindLocalVar(lIdent.Name); v == noValue {
					vl, err := checkSingleValue(mch.evalExpr(ns, r))
					if err != nil {
						return err
					}
					v := reflect.New(vl.Type())
					v.Elem().Set(vl)
					ns.AddLocalVar(lIdent.Name, v)
				}
			}
		} else {
			for i, l := range st.Lhs {
				lV, err := checkSingleValue(mch.evalExpr(ns, l))
				if err != nil {
					return err
				}
				if !lV.CanSet() {
					return fmt.Errorf("Can not assign to %s", l)
				}

				rV, err := mch.evalExpr(ns, st.Rhs[i])
				if err != nil {
					return err
				}

				lV.Set(rV[0])
			}
		}

	case *ast.ExprStmt:
		_, err := mch.evalExpr(ns, st.X)
		return err
		
	default:
		log.Println("Unknown statement type")
		ast.Print(token.NewFileSet(), st)
		return fmt.Errorf("Unknown statement type")
	}
	return nil
}

func isFragmentError(errList scanner.ErrorList, lastLine int) bool {
	err := errList[len(errList)-1]
	return err.Pos.Line == lastLine
}

const (
	srcPrefix = `package main; func main() {
`
	srcSuffix = `
}`
)

func (mch *machine) Run(line string) (isFragment bool) {
	src := srcPrefix + line + srcSuffix

	nLines := len(strings.Split(src, "\n"))

	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, "", src, 0)
	if err != nil {
		isFragment = isFragmentError(err.(scanner.ErrorList), nLines)
		if isFragment {
			return true
		}
		ast.Print(token.NewFileSet(), err)
		log.Printf("Syntax error: %v", err)
		return false
	}
	//	log.Println(line)
	for _, st := range f.Decls[0].(*ast.FuncDecl).Body.List {
		if err := mch.runStatement(mch.GlobalNameSpace, st); err != nil {
			log.Println(err)
			break
		}
	}
	//	log.Println(mch.GlobalNameSpace)
	return false
}

func New() Machine {
	return &machine{
		GlobalNameSpace: newNameSpace(),
		Packages: map[string]map[string]reflect.Value{
			"fmt": map[string]reflect.Value{
				"Println": reflect.ValueOf(fmt.Println),
			},
		},
	}
}
