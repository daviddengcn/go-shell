package gsvm

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
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

func matchDestType(vl reflect.Value, dstTp reflect.Type) reflect.Value {
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

	return noValue, cannotUseAsInAssignmentErr(v, dstTp)
}

var (
	basicTypes = map[string]reflect.Type{
		"int":        reflect.TypeOf(int(0)),
		"int8":       reflect.TypeOf(int8(0)),
		"int16":      reflect.TypeOf(int16(0)),
		"int32":      reflect.TypeOf(int32(0)),
		"rune":       reflect.TypeOf(int32(0)),
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
	}
)

func (mch *machine) evalType(expr ast.Expr) (reflect.Type, error) {
	switch expr := expr.(type) {
	case *ast.Ident:
		tp, ok := basicTypes[expr.Name]
		if ok {
			return tp, nil
		}

		return nil, unknownTypeErr(expr.Name)
	default:
		ast.Print(token.NewFileSet(), expr)
		return nil, fmt.Errorf("Unknown type expr: %s", expr)
	}
}
