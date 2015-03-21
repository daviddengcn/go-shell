package gsvm

import (
	"fmt"
	"go/ast"
	"reflect"
)

func redeclareVarErr(name string) error {
	return fmt.Errorf("%s redeclare in this block", name)
}

func nonBoolAsConditionErr(cnd reflect.Value, st string) error {
	return fmt.Errorf("non-bool %v (type %v) used as %s condition)", cnd.Interface(), cnd.Type(), st)
}

func invalidOperationErr(op string, tp reflect.Type) error {
	return fmt.Errorf("operator %s not defined on %s", op, tp.Name())
}

func cannotAssignToErr(v string) error {
	return fmt.Errorf("Can not assign to %s", v)
}

func cannotUseAsInAssignmentErr(vl reflect.Value, dstTp reflect.Type) error {
	return fmt.Errorf("cannot use %s (type %s) as type %s in assignment", vl, vl.Type(), dstTp)
}

func cannotUseAsInArgumentErr(vl reflect.Value, dstTp reflect.Type, fn string) error {
	return fmt.Errorf("cannot use %s (type %s) as type %s in argument to %s", vl, vl.Type(), dstTp, fn)
}

func unknownTypeErr(name string) error {
	return fmt.Errorf("Unknown type %s", name)
}

func cannotTakeTheAddressOfErr(expr ast.Expr) error {
	return fmt.Errorf("cannot take the address of %v", expr)
}

func invalidIndirectOfErr(vl reflect.Value) error {
	return fmt.Errorf("invalid indirect of %v (type %v)", vl, vl.Type())
}

func mismatchTypesErr(t1, t2 reflect.Type) error {
	return fmt.Errorf("mismatched types %v and %v", t1, t2)
}

func tooManyArgumentsToConversionErr(tp reflect.Type) error {
	return fmt.Errorf("too many arguments to conversion to %v", tp)
}

func missingArgumentToConversionErr(tp reflect.Type) error {
	return fmt.Errorf("missing argument to conversion to %v", tp)
}

func missingArgumentToFuncErr(name string) error {
	return fmt.Errorf("missing argument to %s", name)
}

func cannotConvertToErr(vl reflect.Value, dstTp reflect.Type) error {
	return fmt.Errorf("cannot convert %v (type %v) to type %v", vl, vl.Type(), dstTp)
}

func notEnoughArgumentsErr(fn string) error {
	return fmt.Errorf("not enough arguments in call to %s", fn)
}

func tooManyArgumentsErr(fn string) error {
	return fmt.Errorf("too many arguments in call to %s", fn)
}

type UndefinedError struct {
	error
}

func undefinedErr(s string) error {
	return UndefinedError{fmt.Errorf("undefined: %v", s)}
}

func cannotMakeTypeErr(tp reflect.Type) error {
	return fmt.Errorf("cannot make type %v", tp)
}

func invalidArgumentForFuncErr(vl reflect.Value, fn string) error {
	return fmt.Errorf("invalid argument %v (type %v) for %v", vl, vl.Type(), fn)
}

func notATypeErr(name string) error {
	return fmt.Errorf("%v is not a type", name)
}

func cannotUseAsTypeInErr(vl reflect.Value, tp reflect.Type, pos string) error {
    return fmt.Errorf("cannot use %v (type %v) as type %v in %v", vl, vl.Type(), tp, pos)
}

func FirstArugmentToMustBeHaveErr(fn, expTp string, actTp reflect.Type) error {
	return fmt.Errorf("first argument to %s must be %s; have %v", fn, expTp, actTp)
}

var (
	noNewVarsErr = fmt.Errorf("no new on left side of :=")
)
