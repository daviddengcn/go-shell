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

var (
	trueValue = reflect.ValueOf(true)
	falseValue = reflect.ValueOf(false)
)

func keywordValue(ident string) reflect.Value {
	switch ident {
	case "true": return trueValue
	case "false": return falseValue
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
		if v := keywordValue(expr.Name); v != noValue {
			return fromSingleValue(v, nil)
		}
	
		if v := ns.FindVar(expr.Name); v != noValue {
			return fromSingleValue(reflect.Indirect(v), nil)
		}
		
		return nil, fmt.Errorf("Unknown Ident %v", expr.Name)

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
		
	case *ast.BinaryExpr:
		x, err := checkSingleValue(mch.evalExpr(ns, expr.X))
		if err != nil {
			return nil, err
		}
		
		if expr.Op == token.LAND || expr.Op == token.LOR {
			// short cut for boolean
			if x.Kind() != reflect.Bool {
				return nil, invalidOperationErr(expr.Op.String(), x.Type())
			}
			bX := x.Bool()
			if expr.Op == token.LAND && !bX || expr.Op == token.LOR && bX {
				return fromSingleValue(reflect.ValueOf(bX), nil)
			}
			
			y, err := checkSingleValue(mch.evalExpr(ns, expr.Y))
			return fromSingleValue(y, err)
		}
		
		y, err := checkSingleValue(mch.evalExpr(ns, expr.Y))
		if err != nil {
			return nil, err
		}
		
		
		if x, y, err = matchType(x, y); err != nil {
			return nil, err
		}
		
		switch expr.Op {
		case token.LSS:
			switch x.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return fromSingleValue(reflect.ValueOf(x.Int() < y.Int()), nil)
			default:
				panic("")
			}
			
		case token.ADD:
			switch x.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				res := reflect.New(x.Type())
				res.Elem().SetInt(x.Int() + y.Int())
				return fromSingleValue(res.Elem(), nil)
			default:
				panic("")
			}
		
		default:
			return nil, fmt.Errorf("Unknown op: %v", expr.Op)
		}
		
	default:
		log.Println("Unknown expr type")
		ast.Print(token.NewFileSet(), expr)
		return []reflect.Value{reflect.ValueOf(expr)}, nil
	}
}

func (mch *machine) typeOf(tp ast.Expr) (reflect.Type, error) {
	switch tp := tp.(type) {
	case *ast.Ident:
		switch tp.Name {
		case "int": return reflect.TypeOf(int(0)), nil
		case "int8": return reflect.TypeOf(int8(0)), nil
		case "int16": return reflect.TypeOf(int16(0)), nil
		case "int32", "rune": return reflect.TypeOf(int32(0)), nil
		case "int64": return reflect.TypeOf(int64(0)), nil
		case "uint": return reflect.TypeOf(uint(0)), nil
		case "uint8", "byte": return reflect.TypeOf(uint8(0)), nil
		case "uint16": return reflect.TypeOf(uint16(0)), nil
		case "uint32": return reflect.TypeOf(uint32(0)), nil
		case "uint64": return reflect.TypeOf(uint64(0)), nil
		case "string": return reflect.TypeOf(""), nil
		default:
			return nil, fmt.Errorf("Unknown type %s", tp.Name)
		}
	default:
		return nil, fmt.Errorf("Unknown type expr: %s", tp)
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
			// FIXME should be parallel assignment
			hasNew := false
			for _, l := range st.Lhs {
				lIdent := l.(*ast.Ident)
				if ns.FindLocalVar(lIdent.Name) == noValue {
					hasNew = true
				}
			}
			if !hasNew {
				return fmt.Errorf("no new on left side of :=")
			}

			values := make([]reflect.Value, len(st.Rhs))
			for i, r := range st.Rhs {
				rV, err := checkSingleValue(mch.evalExpr(ns, r))
				if err != nil {
					return err
				}
				if len(st.Rhs) > 1 && rV.CanAddr() {
					// Make a copy of lvalue for parallel assignments
					tmp := reflect.New(rV.Type())
					tmp.Elem().Set(rV)
					rV = tmp.Elem()
				}
				values[i] = rV
			}
			
			for i, l := range st.Lhs {
				lIdent := l.(*ast.Ident)
				v := ns.FindLocalVar(lIdent.Name)
				if v == noValue {
					v = reflect.New(values[i].Type())
					ns.AddLocalVar(lIdent.Name, v)
				}
				
				v.Elem().Set(values[i])
			}
		} else {
			values := make([]reflect.Value, len(st.Rhs))
			for i, r := range st.Rhs {
				rV, err := checkSingleValue(mch.evalExpr(ns, r))
				if err != nil {
					return err
				}
				if len(st.Rhs) > 1 && rV.CanAddr() {
					// Make a copy of lvalue for parallel assignments
					tmp := reflect.New(rV.Type())
					tmp.Elem().Set(rV)
					rV = tmp.Elem()
				}
				values[i] = rV
			}
			for i, l := range st.Lhs {
				lV, err := checkSingleValue(mch.evalExpr(ns, l))
				if err != nil {
					return err
				}
				if !lV.CanSet() {
					return cannotAssignToErr(lV.String())
				}
				lV.Set(values[i])
			}
		}

	case *ast.ExprStmt:
		_, err := mch.evalExpr(ns, st.X)
		return err
		
	case *ast.DeclStmt:
		switch decl := st.Decl.(type) {
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				spec := spec.(*ast.ValueSpec)
				var values []reflect.Value
				if len(spec.Values) == 1 {
					var err error
					values, err = mch.evalExpr(ns, spec.Values[0])
					if err != nil {
						return err
					}
				} else if len(spec.Values) > 0 {
					values = make([]reflect.Value, len(spec.Values))
					for i, valueExpr := range spec.Values {
						value, err := checkSingleValue(mch.evalExpr(ns, valueExpr))
						if err != nil {
							return err
						}
						values[i] = value
					}
				} else if spec.Type == nil {
					return fmt.Errorf("Need type")
				}
				
				if values != nil && len(spec.Names) != len(values) {
					return fmt.Errorf("assignment count mismatch: %d = %d", len(spec.Names), len(values))
				}
				
				for i, name := range spec.Names {
					if v := ns.FindLocalVar(name.Name); v != noValue {
						return redeclareVarErr(name.Name)
					}
					var v reflect.Value
					if spec.Type != nil {
						tp, err := mch.typeOf(spec.Type)
						if err != nil {
							return err
						}
						v = reflect.New(tp)
					} else {
						v = reflect.New(values[i].Type())
					}
					
					if values != nil {
						v.Elem().Set(values[i])
					}
					ns.AddLocalVar(name.Name, v)
				}
			}
		
		case *ast.FuncDecl:
			// TODO
			ast.Print(token.NewFileSet(), decl)
			return nil
		}
		
	case *ast.BlockStmt:
		blkNs := ns.NewBlock()
		for _, st := range st.List {
			if err := mch.runStatement(blkNs, st); err != nil {
				return err
			}
		}
		
	case *ast.ForStmt:
		blkNs := ns.NewBlock()
		if st.Init != nil {
			mch.runStatement(blkNs, st.Init)
		}
		
		for {
			cond := true
			if st.Cond != nil {
				cnd, err := checkSingleValue(mch.evalExpr(blkNs, st.Cond))
				if err != nil {
					return err
				}
				
				if cnd.Kind() != reflect.Bool {
					return nonBoolAsConditionErr(cnd, "for")
				}
				cond = cnd.Bool()
			}
			if !cond {
				break
			}
			
			if err := mch.runStatement(blkNs, st.Body); err != nil {
				return err
			}
			
			if st.Post != nil {
				if err := mch.runStatement(blkNs, st.Post); err != nil {
					return err
				}
			}
		}
		
	case *ast.IncDecStmt:
		x, err := checkSingleValue(mch.evalExpr(ns, st.X))
		if err != nil {
			return err
		}
		
		if !x.CanSet() {
			return cannotAssignToErr(x.String())
		}
		
		switch  x.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if st.Tok == token.INC {
				x.SetInt(x.Int() + 1)
			} else {
				x.SetInt(x.Int() - 1)
			}
			return nil
			
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if st.Tok == token.INC {
				x.SetUint(x.Uint() + 1)
			} else {
				x.SetUint(x.Uint() - 1)
			}
			return nil
			
		case reflect.Float32, reflect.Float64:
			if st.Tok == token.INC {
				x.SetFloat(x.Float() + 1)
			} else {
				x.SetFloat(x.Float() - 1)
			}
			return nil

		case reflect.Complex64, reflect.Complex128:			
			if st.Tok == token.INC {
				x.SetComplex(x.Complex() + 1)
			} else {
				x.SetComplex(x.Complex() - 1)
			}
			return nil
			
		default:
			return invalidOperationErr(st.Tok.String(), x.Type())
		}
		panic("")
		
	default:
		log.Println("Unknown statement type")
		ast.Print(token.NewFileSet(), st)
		return fmt.Errorf("Unknown statement type")
	}
	return nil
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
		log.Printf("Syntax error: %v %d", err, nLines)
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
