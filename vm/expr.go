package gsvm

import (
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"reflect"
	"strconv"
	"unicode/utf8"
)

func checkSingleValue(vls []reflect.Value, err error) (reflect.Value, error) {
	if err != nil {
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

func valueToResult(vl interface{}) ([]reflect.Value, error) {
	return []reflect.Value{reflect.ValueOf(vl)}, nil
}

func typedValueToResult(vl interface{}, tp reflect.Type) ([]reflect.Value, error) {
	return []reflect.Value{reflect.ValueOf(vl).Convert(tp)}, nil
}

// Returns slice of values themselves not the pointers.
func (mch *machine) evalExpr(ns NameSpace, expr ast.Expr) ([]reflect.Value, error) {
	switch expr := expr.(type) {
	case *ast.BasicLit:
		switch expr.Kind {
		case token.INT:
			var v intLiteral
			fmt.Sscan(expr.Value, &v)
			return valueToResult(v)

		case token.IMAG:
			var v floatLiteral
			fmt.Sscan(expr.Value, &v)
			return valueToResult(complexLiteral(complex(0, v)))

		case token.CHAR:
			s, _ := strconv.Unquote(expr.Value)
			v, _ := utf8.DecodeRuneInString(s)
			return valueToResult(runeLiteral(v))

		case token.STRING:
			v, _ := strconv.Unquote(expr.Value)
			return valueToResult(stringLiteral(v))

		case token.FLOAT:
			v, _ := strconv.ParseFloat(expr.Value, 64)
			return valueToResult(floatLiteral(v))
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
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return fromSingleValue(reflect.ValueOf(x.Uint() < y.Uint()), nil)
			case reflect.Float32, reflect.Float64:
				return fromSingleValue(reflect.ValueOf(x.Float() < y.Float()), nil)
			case reflect.String:
				return fromSingleValue(reflect.ValueOf(x.String() < y.String()), nil)
			}
		case token.LEQ:
			switch x.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return fromSingleValue(reflect.ValueOf(x.Int() <= y.Int()), nil)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return fromSingleValue(reflect.ValueOf(x.Uint() <= y.Uint()), nil)
			case reflect.Float32, reflect.Float64:
				return fromSingleValue(reflect.ValueOf(x.Float() <= y.Float()), nil)
			case reflect.String:
				return fromSingleValue(reflect.ValueOf(x.String() <= y.String()), nil)
			}
		case token.GTR:
			switch x.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return fromSingleValue(reflect.ValueOf(x.Int() > y.Int()), nil)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return fromSingleValue(reflect.ValueOf(x.Uint() > y.Uint()), nil)
			case reflect.Float32, reflect.Float64:
				return fromSingleValue(reflect.ValueOf(x.Float() > y.Float()), nil)
			case reflect.String:
				return fromSingleValue(reflect.ValueOf(x.String() > y.String()), nil)
			}
		case token.GEQ:
			switch x.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return fromSingleValue(reflect.ValueOf(x.Int() >= y.Int()), nil)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return fromSingleValue(reflect.ValueOf(x.Uint() >= y.Uint()), nil)
			case reflect.Float32, reflect.Float64:
				return fromSingleValue(reflect.ValueOf(x.Float() >= y.Float()), nil)
			case reflect.String:
				return fromSingleValue(reflect.ValueOf(x.String() >= y.String()), nil)
			}

		case token.EQL:
			return fromSingleValue(reflect.ValueOf(x.Interface() == y.Interface()), nil)
		case token.NEQ:
			return fromSingleValue(reflect.ValueOf(x.Interface() != y.Interface()), nil)

		case token.ADD:
			switch x.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return typedValueToResult(x.Int()+y.Int(), x.Type())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return typedValueToResult(x.Uint()+y.Uint(), x.Type())
			case reflect.Float32, reflect.Float64:
				return typedValueToResult(x.Float()+y.Float(), x.Type())
			case reflect.Complex64, reflect.Complex128:
				return typedValueToResult(x.Complex()+y.Complex(), x.Type())
			}

		default:
			return nil, fmt.Errorf("Unknown op: %v", expr.Op)
		}

		return nil, invalidOperationErr(expr.Op.String(), x.Type())
	default:
		log.Println("Unknown expr type")
		ast.Print(token.NewFileSet(), expr)
		return []reflect.Value{reflect.ValueOf(expr)}, nil
	}
	return nil, fmt.Errorf("Unknown expr type")
}
