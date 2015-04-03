package gsvm

import (
	"go/ast"
	"go/token"
	"reflect"

	"github.com/daviddengcn/go-villa"
)

type intLiteral int64
type floatLiteral float64
type complexLiteral complex128
type runeLiteral rune
type stringLiteral string

var (
	intLiteralType     = reflect.TypeOf(intLiteral(0))
	floatLiteralType   = reflect.TypeOf(floatLiteral(0))
	complexLiteralType = reflect.TypeOf(complexLiteral(0))
	runeLiteralType    = reflect.TypeOf(runeLiteral(0))
	stringLiteralType  = reflect.TypeOf(stringLiteral(""))
)

func removeBasicLit(vl reflect.Value) reflect.Value {
	switch vl.Type() {
	case intLiteralType:
		return reflect.ValueOf(int(vl.Int()))
	case floatLiteralType:
		return reflect.ValueOf(float64(vl.Float()))
	case complexLiteralType:
		return reflect.ValueOf(complex128(vl.Complex()))
	case runeLiteralType:
		return reflect.ValueOf(rune(vl.Int()))
	case stringLiteralType:
		return reflect.ValueOf(vl.String())
	case MapIndexValueType:
		vl := vl.Interface().(MapIndexValue)
		return removeBasicLit(vl.X.MapIndex(vl.Key))
	}

	return vl
}

var (
	intAssignableTo = [...]bool{
		reflect.Int:     true,
		reflect.Int8:    true,
		reflect.Int16:   true,
		reflect.Int32:   true,
		reflect.Int64:   true,
		reflect.Uint:    true,
		reflect.Uint8:   true,
		reflect.Uint16:  true,
		reflect.Uint32:  true,
		reflect.Uint64:  true,
		reflect.Float32: true,
		reflect.Float64: true,
	}

	floatAssignableTo = [...]bool{
		reflect.Float32: true,
		reflect.Float64: true,
	}

	complexAssignableTo = [...]bool{
		reflect.Complex64:  true,
		reflect.Complex128: true,
	}
)

func matchType(x, y reflect.Value) (nX, nY reflect.Value, err error) {
	if x.Type() == y.Type() {
		return x, y, nil
	}

	if x.Type() == runeLiteralType {
		x = matchDestType(x, y.Type())
	} else if y.Type() == runeLiteralType {
		y = matchDestType(y, x.Type())
	} else {
		x, y = matchDestType(x, y.Type()), matchDestType(y, x.Type())
	}

	if x.Type() == y.Type() {
		return x, y, nil
	}

	return x, y, mismatchTypesErr(x.Type(), y.Type())
}

// matchDestType tries match vl with dstTp and return converted value. If
// fail to match, return vl.
func matchDestType(vl reflect.Value, dstTp reflect.Type) reflect.Value {
	if vl.Type() == MapIndexValueType {
		vl := vl.Interface().(MapIndexValue)
		return matchDestType(vl.X.MapIndex(vl.Key), dstTp)
	}

	if vl.Type() == ConstValueType {
		vl = vl.Field(0).Interface().(reflect.Value)
	}

	if vl.Type() == dstTp {
		return vl
	}

	canConvert := false
	switch vl.Type() {
	case intLiteralType, runeLiteralType:
		canConvert = int(dstTp.Kind()) < len(intAssignableTo) && intAssignableTo[dstTp.Kind()]

	case floatLiteralType:
		canConvert = int(dstTp.Kind()) < len(floatAssignableTo) && floatAssignableTo[dstTp.Kind()]

	case complexLiteralType:
		canConvert = int(dstTp.Kind()) < len(complexAssignableTo) && complexAssignableTo[dstTp.Kind()]

	case stringLiteralType:
		canConvert = dstTp.Kind() == reflect.String
	}

	if canConvert {
		return vl.Convert(dstTp)
	}

	// try convert int/real number to complex
	switch vl.Type() {
	case intLiteralType, runeLiteralType:
		switch dstTp.Kind() {
		case reflect.Complex64:
			vl = reflect.ValueOf(complex(float32(vl.Int()), 0))
		case reflect.Complex128:
			vl = reflect.ValueOf(complex(float64(vl.Int()), 0))
		}

	case floatLiteralType:
		switch dstTp.Kind() {
		case reflect.Complex64:
			vl = reflect.ValueOf(complex(float32(vl.Float()), 0))
		case reflect.Complex128:
			vl = reflect.ValueOf(complex(float64(vl.Float()), 0))
		}
	}

	return vl
}

func literalAssignConvert(v reflect.Value, dstTp reflect.Type) (reflect.Value, error) {
	if v.Type() == dstTp {
		return v, nil
	}
	convertable := false
	switch v.Type() {
	case intLiteralType:
		convertable = int(dstTp.Kind()) < len(intAssignableTo) && intAssignableTo[dstTp.Kind()]

	case floatLiteralType:
		convertable = int(dstTp.Kind()) < len(floatAssignableTo) && floatAssignableTo[dstTp.Kind()]

	case complexLiteralType:
		convertable = int(dstTp.Kind()) < len(complexAssignableTo) && complexAssignableTo[dstTp.Kind()]

	case stringLiteralType:
		convertable = dstTp.Kind() == reflect.String
	}

	if convertable {
		return v.Convert(dstTp), nil
	}

	return NoValue, cannotUseAsInAssignmentErr(v, dstTp)
}

var (
	intType  = reflect.TypeOf(int(0))
	runeType = reflect.TypeOf(rune(0))
)

var (
	basicTypes = map[string]reflect.Type{
		"int":        intType,
		"int8":       reflect.TypeOf(int8(0)),
		"int16":      reflect.TypeOf(int16(0)),
		"int32":      reflect.TypeOf(int32(0)),
		"rune":       runeType,
		"int64":      reflect.TypeOf(int64(0)),
		"uint":       reflect.TypeOf(uint(0)),
		"uint8":      reflect.TypeOf(uint8(0)),
		"byte":       reflect.TypeOf(uint8(0)),
		"uint16":     reflect.TypeOf(uint16(0)),
		"uint32":     reflect.TypeOf(uint32(0)),
		"uint64":     reflect.TypeOf(uint64(0)),
		"float32":    reflect.TypeOf(float32(0)),
		"float64":    reflect.TypeOf(float64(0)),
		"complex64":  reflect.TypeOf(complex64(0)),
		"complex128": reflect.TypeOf(complex128(0)),
		"string":     reflect.TypeOf(""),
		"error":      reflect.TypeOf((*error)(nil)).Elem(),
	}
)

// Holding a constant value
type ConstValue struct {
	reflect.Value
}

var ConstValueType = reflect.TypeOf(ConstValue{})

func ToConstant(vl reflect.Value) reflect.Value {
	return reflect.ValueOf(ConstValue{vl})
}

// Holding a type value
type TypeValue struct {
	reflect.Type
}

func PtrToTypeValue(ptr interface{}) reflect.Value {
	return reflect.ValueOf(TypeValue{reflect.TypeOf(ptr).Elem()})
}

var TypeValueType = reflect.TypeOf(TypeValue{})

type MapIndexValue struct {
	// a map Value
	X reflect.Value
	// a value same typed with X.Type().Key()
	Key reflect.Value
}

var MapIndexValueType = reflect.TypeOf(MapIndexValue{})

var chanDir = map[ast.ChanDir]reflect.ChanDir{
	ast.SEND: reflect.SendDir,
	ast.RECV: reflect.RecvDir,
	ast.SEND | ast.RECV: reflect.BothDir,
}

func (mch *machine) evalType(ns NameSpace, expr ast.Expr) (reflect.Type, error) {
	switch expr := expr.(type) {
	case *ast.Ident:
		tp, ok := basicTypes[expr.Name]
		if ok {
			return tp, nil
		}

		return nil, unknownTypeErr(expr.Name)
	case *ast.ArrayType:
		if expr.Len == nil {
			elTp, err := mch.evalType(ns, expr.Elt)
			if err != nil {
				return nil, err
			}
			return reflect.SliceOf(elTp), nil
		}
		ast.Print(token.NewFileSet(), expr)
		return nil, villa.Errorf("Wait for reflect.ArrayOf")

	case *ast.SelectorExpr:
		x, err := checkSingleValue(mch.evalExpr(ns, expr.X))
		if err != nil {
			return nil, err
		}

		switch x.Type() {
		case PackageType:
			x := x.Interface().(Package)
			if vl, ok := x[expr.Sel.Name]; ok {
				if vl.Type() != TypeValueType {
					return nil, notATypeErr(expr.Sel.Name)
				}
				return vl.Interface().(TypeValue).Type, nil
			}
			return nil, nil
		default:
			ast.Print(token.NewFileSet(), expr)
			return nil, villa.Errorf("Unknown type expr: SelectorExpr X: %v", x.Type())
		}
		ast.Print(token.NewFileSet(), expr)
		return nil, villa.Errorf("Unknown type expr: %+v", expr)

	case *ast.MapType:
		keyTp, err := mch.evalType(ns, expr.Key)
		if err != nil {
			return nil, err
		}

		valTp, err := mch.evalType(ns, expr.Value)
		if err != nil {
			return nil, err
		}

		// TODO check keyTp
		return reflect.MapOf(keyTp, valTp), nil

	case *ast.FuncType:
		return nil, villa.Error("Waiting for reflect package to support reflect.FuncOf")
		
	case *ast.ChanType:
		vType, err := mch.evalType(ns, expr.Value)
		if err != nil {
			return nil, err
		}
		
		return reflect.ChanOf(chanDir[expr.Dir], vType), nil
	default:
		ast.Print(token.NewFileSet(), expr)
		return nil, villa.Errorf("Unknown type expr: %+v", expr)
	}
}
