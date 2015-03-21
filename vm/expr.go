package gsvm

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"strconv"
	"unicode/utf8"
	
	"github.com/daviddengcn/go-villa"
)

func checkSingleValue(vls []reflect.Value, err error) (reflect.Value, error) {
	if err != nil {
		return NoValue, err
	}
	if len(vls) != 1 {
		return NoValue, fmt.Errorf("multiple-value(%d) in single-value context", len(vls))
	}
	return vls[0], nil
}

func fromSingleValue(vl reflect.Value, err error) ([]reflect.Value, error) {
	if err != nil {
		return nil, err
	}
	return []reflect.Value{vl}, nil
}

func singleValue(vl reflect.Value) ([]reflect.Value, error) {
	return []reflect.Value{vl}, nil
}

func valueToResult(vl interface{}) ([]reflect.Value, error) {
	return []reflect.Value{reflect.ValueOf(vl)}, nil
}

func typedValueToResult(vl interface{}, tp reflect.Type) ([]reflect.Value, error) {
	return []reflect.Value{reflect.ValueOf(vl).Convert(tp)}, nil
}

func calcFuncInNumRange(tp reflect.Type) (mn, mx int) {
	if tp.IsVariadic() {
		return tp.NumIn() - 1, -1
	}
	return tp.NumIn(), tp.NumIn()
}

func valueEqual(a, b reflect.Value) (bool, error) {
	if !a.Type().Comparable() {
		return false, fmt.Errorf("%v not comparable", a)
	}
	if !b.Type().Comparable() {
		return false, fmt.Errorf("%v not comparable", b)
	}

	return a.Interface() == b.Interface(), nil
}

func asInteger(vl reflect.Value) (int, error) {
	return int(vl.Int()), nil
}

type builtinFuncImpl func(mch *machine, ns NameSpace, args []ast.Expr) ([]reflect.Value, error)

var gBuiltinFuncs map[string]builtinFuncImpl

func init() {
	gBuiltinFuncs = map[string]builtinFuncImpl {
		"make": func(mch *machine, ns NameSpace, args []ast.Expr) ([]reflect.Value, error) {
			if len(args) == 0 {
				return nil, missingArgumentToFuncErr("make")
			}
			tp, err := mch.evalType(ns, args[0])
			if err != nil {
				return nil, err
			}
			
			switch tp.Kind() {
			case reflect.Slice:
				if len(args) > 3 {
					return nil, tooManyArgumentsErr("make")
				}
				l, err := checkSingleValue(mch.evalExpr(ns, args[1]))
				if err != nil {
					return nil, err
				}
				
				ln, err := asInteger(l)
				if err != nil {
					return nil, err
				}
				cp := ln
				if len(args) == 3 {
					c, err := checkSingleValue(mch.evalExpr(ns, args[2]))
					if err != nil {
						return nil, err
					}
					
					cp, err = asInteger(c)
					if err != nil {
						return nil, err
					}
				}
				
				return singleValue(reflect.MakeSlice(tp, ln, cp))
			// TODO make(map)
			default:
				return nil, cannotMakeTypeErr(tp)
			}
			return nil, nil
		},
		"len": func(mch *machine, ns NameSpace, args []ast.Expr) ([]reflect.Value, error) {
			if len(args) < 1 {
				return nil, missingArgumentToFuncErr("len")
			}
			if len(args) > 1 {
				return nil, tooManyArgumentsErr("len")
			}
			
			vl, err := checkSingleValue(mch.evalExpr(ns, args[0]))
			if err != nil {
				return nil, err
			}
			
			switch vl.Kind() {
			case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
				return valueToResult(intLiteral(vl.Len()))
			default:
				return nil, invalidArgumentForFuncErr(vl, "len")
			}
		},
		"append": func(mch *machine, ns NameSpace, args []ast.Expr) ([]reflect.Value, error) {
			if len(args) < 2 {
				return nil, missingArgumentToFuncErr("append")
			}

			x, err := checkSingleValue(mch.evalExpr(ns, args[0]))
			if err != nil {
				return nil, err
			}
			
			if (x.Kind() != reflect.Slice) {
				return nil, ArugmentToMustBeHaveErr("first", "append", "slice", x.Type())
			}
			
			args = args[1:]

			els := make([]reflect.Value, len(args))
			for i, arg := range args {
				argV, err := checkSingleValue(mch.evalExpr(ns, arg))
				if err != nil {
					return nil, err
				}
				els[i] = matchDestType(argV, x.Type().Elem())
			}
			
			return singleValue(reflect.Append(x, els...))
		},
		"copy": func(mch *machine, ns NameSpace, args []ast.Expr) ([]reflect.Value, error) {
			if len(args) < 2 {
				return nil, missingArgumentToFuncErr("len")
			}
			if len(args) > 2 {
				return nil, tooManyArgumentsErr("len")
			}
			
			x, err := checkSingleValue(mch.evalExpr(ns, args[0]))
			if err != nil {
				return nil, err
			}
			if (x.Kind() != reflect.Slice) {
				return nil, ArugmentToMustBeHaveErr("first", "copy", "slice", x.Type())
			}
			
			y, err := checkSingleValue(mch.evalExpr(ns, args[1]))
			if err != nil {
				return nil, err
			}
			if (y.Kind() != reflect.Slice) {
				return nil, ArugmentToMustBeHaveErr("second", "copy", "slice", y.Type())
			}
			
			return valueToResult(reflect.Copy(x, y))
		}
	}
}

func builtinFunc(fun ast.Expr) string {
	switch fun := fun.(type) {
	case *ast.Ident:
		if _, ok := gBuiltinFuncs[fun.Name]; ok {
			return fun.Name
		}
	}
	return ""
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
		if v := keywordValue(expr.Name); v != NoValue {
			return fromSingleValue(v, nil)
		}

		if v := ns.Find(expr.Name); v != NoValue {
			return fromSingleValue(v, nil)
		}

		if tp, err := mch.evalType(ns, expr); err == nil {
			return singleValue(reflect.ValueOf(TypeValue{tp}))
		}

		return nil, undefinedErr(expr.Name)

	case *ast.CallExpr:
		fn, err := checkSingleValue(mch.evalExpr(ns, expr.Fun))
		if err != nil {
			if _, ok := err.(UndefinedError); ok {
				fun := builtinFunc(expr.Fun)
				if fun == "" {
					return nil, err
				}
				return gBuiltinFuncs[fun](mch, ns, expr.Args)
			}
			return nil, err
		}
		fnType := fn.Type()

		if fnType == TypeValueType {
			tp := fn.Interface().(TypeValue).Type
			if len(expr.Args) > 1 {
				return nil, tooManyArgumentsToConversionErr(tp)
			}
			if len(expr.Args) < 1 {
				return nil, missingArgumentToConversionErr(tp)
			}
			v, err := checkSingleValue(mch.evalExpr(ns, expr.Args[0]))
			if err != nil {
				return nil, err
			}

			if !v.Type().ConvertibleTo(tp) {
				return nil, cannotConvertToErr(v, tp)
			}
			return singleValue(v.Convert(tp))
		}

		if fn.Kind() != reflect.Func {
			return nil, fmt.Errorf("cannot call non-function (type %s)", fn.Type())
		}

		var args []reflect.Value
		if len(expr.Args) == 1 {
			// actually input args number is the number of return values
			var err error
			if args, err = mch.evalExpr(ns, expr.Args[0]); err != nil {
				return nil, err
			}
		} else {
			args = make([]reflect.Value, len(expr.Args))
			for i, arg := range expr.Args {
				argV, err := checkSingleValue(mch.evalExpr(ns, arg))
				if err != nil {
					return nil, err
				}
				args[i] = argV
			}
		}

		mn, mx := calcFuncInNumRange(fnType)
		if len(args) < mn {
			return nil, notEnoughArgumentsErr(fn.String())
		}

		if mx >= 0 && len(args) > mx {
			return nil, tooManyArgumentsErr(fn.String())
		}

		for i := 0; i < mn; i++ {
			tp := fnType.In(i)
			args[i] = removeBasicLit(matchDestType(args[i], tp))
			if !args[i].Type().AssignableTo(tp) {
				return nil, cannotUseAsInArgumentErr(args[i], tp, fn.String())
			}
		}

		if fn.Type().IsVariadic() {
			tp := fnType.In(fnType.NumIn() - 1).Elem()
			for i := mn; i < len(args); i++ {
				args[i] = removeBasicLit(matchDestType(args[i], tp))
				if !args[i].Type().AssignableTo(tp) {
					return nil, cannotUseAsInArgumentErr(args[i], tp, fn.String())
				}
			}
		}

		return fn.Call(args), nil

	case *ast.SelectorExpr:
		x, err := checkSingleValue(mch.evalExpr(ns, expr.X))
		if err != nil {
			return nil, err
		}

		switch x.Type() {
		case ConstValueType:
		case TypeValueType:
		case PackageType:
			x := x.Interface().(Package)
			if vl, ok := x[expr.Sel.Name]; ok {
				return singleValue(vl)
			}
			return nil, undefinedErr(fmt.Sprintf("%v.%v", expr.X, expr.Sel.Name))
		default:
		}

		for x.Kind() == reflect.Ptr {
			x = x.Elem()
		}
		
		if x.Kind() != reflect.Struct {
			return nil, fmt.Errorf("no-struct")
		}
		
		if vl := x.FieldByName(expr.Sel.Name); vl.IsValid() {
			return singleValue(vl)
		}
		if vl := x.MethodByName(expr.Sel.Name); vl.IsValid() {
			return singleValue(vl)
		}
		return nil, fmt.Errorf("nofound")

	case *ast.UnaryExpr:
		x, err := checkSingleValue(mch.evalExpr(ns, expr.X))
		if err != nil {
			return nil, err
		}

		switch expr.Op {
		case token.ADD:
			switch x.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return valueToResult(x.Interface())
			}
		case token.SUB:
			switch x.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return typedValueToResult(-x.Int(), x.Type())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return typedValueToResult(-x.Uint(), x.Type())
			}
		case token.NOT:
			switch x.Kind() {
			case reflect.Bool:
				return typedValueToResult(!x.Bool(), x.Type())
			}
		case token.XOR:
			switch x.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return typedValueToResult(^x.Int(), x.Type())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return typedValueToResult(^x.Uint(), x.Type())
			}
		case token.AND:
			if x.CanAddr() {
				return singleValue(x.Addr())
			}
			return nil, cannotTakeTheAddressOfErr(expr.X)
			// TODO token.ARROW
		}
		return nil, invalidOperationErr(expr.Op.String(), x.Type())

	case *ast.StarExpr:
		x, err := checkSingleValue(mch.evalExpr(ns, expr.X))
		if err != nil {
			return nil, err
		}

		if x.Kind() == reflect.Ptr {
			return singleValue(x.Elem())
		}
		return nil, invalidIndirectOfErr(x)

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
			case reflect.String:
				return typedValueToResult(x.String()+y.String(), x.Type())
			}
		case token.SUB:
			switch x.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return typedValueToResult(x.Int()-y.Int(), x.Type())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return typedValueToResult(x.Uint()-y.Uint(), x.Type())
			case reflect.Float32, reflect.Float64:
				return typedValueToResult(x.Float()-y.Float(), x.Type())
			case reflect.Complex64, reflect.Complex128:
				return typedValueToResult(x.Complex()-y.Complex(), x.Type())
			}
		case token.MUL:
			switch x.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return typedValueToResult(x.Int()*y.Int(), x.Type())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return typedValueToResult(x.Uint()*y.Uint(), x.Type())
			case reflect.Float32, reflect.Float64:
				return typedValueToResult(x.Float()*y.Float(), x.Type())
			case reflect.Complex64, reflect.Complex128:
				return typedValueToResult(x.Complex()*y.Complex(), x.Type())
			}
		case token.QUO:
			switch x.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return typedValueToResult(x.Int()/y.Int(), x.Type())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return typedValueToResult(x.Uint()/y.Uint(), x.Type())
			case reflect.Float32, reflect.Float64:
				return typedValueToResult(x.Float()/y.Float(), x.Type())
			case reflect.Complex64, reflect.Complex128:
				return typedValueToResult(x.Complex()/y.Complex(), x.Type())
			}
		case token.REM:
			switch x.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return typedValueToResult(x.Int()%y.Int(), x.Type())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return typedValueToResult(x.Uint()%y.Uint(), x.Type())
			}
		case token.AND:
			switch x.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return typedValueToResult(x.Int()&y.Int(), x.Type())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return typedValueToResult(x.Uint()&y.Uint(), x.Type())
			}
		case token.OR:
			switch x.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return typedValueToResult(x.Int()|y.Int(), x.Type())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return typedValueToResult(x.Uint()|y.Uint(), x.Type())
			}
		case token.XOR:
			switch x.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return typedValueToResult(x.Int()^y.Int(), x.Type())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return typedValueToResult(x.Uint()^y.Uint(), x.Type())
			}
		case token.SHL:
			switch x.Kind() {
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return typedValueToResult(x.Uint()<<y.Uint(), x.Type())
			}
		case token.SHR:
			switch x.Kind() {
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return typedValueToResult(x.Uint()>>y.Uint(), x.Type())
			}
		case token.AND_NOT:
			switch x.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return typedValueToResult(x.Int()&^y.Int(), x.Type())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return typedValueToResult(x.Uint()&^y.Uint(), x.Type())
			}

		default:
			return nil, villa.Errorf("Unknown op: %v", expr.Op)
		}

		return nil, invalidOperationErr(expr.Op.String(), x.Type())
		
	case *ast.IndexExpr:
		x, err := checkSingleValue(mch.evalExpr(ns, expr.X))
		if err != nil {
			return nil, err
		}
		
		index, err := checkSingleValue(mch.evalExpr(ns, expr.Index))
		i, err := asInteger(index)
		if err != nil {
			return nil, err
		}
		
		return valueToResult(IndexType{x, i})
		
	case *ast.CompositeLit:
		tp, err := mch.evalType(ns, expr.Type)
		if err != nil {
			return nil, err
		}
		
		switch tp.Kind() {
		case reflect.Slice:
			vl := reflect.MakeSlice(tp, len(expr.Elts), len(expr.Elts))
			for i, elt := range expr.Elts {
				vlElt, err := checkSingleValue(mch.evalExpr(ns, elt))
				if err != nil {
					return nil, err
				}
				dstElt := matchDestType(vlElt, tp.Elem())
				if !dstElt.Type().AssignableTo(tp.Elem()) {
					return nil, cannotUseAsTypeInErr(dstElt, tp.Elem(), "array element")
				}
				vl.Index(i).Set(dstElt)
			}
			return singleValue(vl)
		default:
			ast.Print(token.NewFileSet(), expr)
			return nil, villa.Errorf("Unknown CompositeLit expr Kind")
		}
	}
	ast.Print(token.NewFileSet(), expr)
	return nil, villa.Errorf("Unknown expr type")
}
