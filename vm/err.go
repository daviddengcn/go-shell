package gsvm

import (
	"fmt"
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

func unknownTypeErr(name string) error {
	return fmt.Errorf("Unknown type %s", name)
}

var (
	noNewVarsErr = fmt.Errorf("no new on left side of :=")
)
