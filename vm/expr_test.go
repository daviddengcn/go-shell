package gsvm

import (
	"reflect"
	"testing"

	"github.com/daviddengcn/go-assert"
)

func TestBinaryExpr(t *testing.T) {
	mch := newMachine()

	assert.NoError(t, mch.Run(`fmt.Println("go" + "lang")`))
	assert.NoError(t, mch.Run(`fmt.Println("7.0/3.0 =", 7.0/3.0)`))
}

func TestUnaryExpr(t *testing.T) {
	mch := newMachine()

	assert.NoError(t, mch.Run(`fmt.Println(!true)`))
	assert.NoError(t, mch.Run(`i := !true`))
	i := mch.GlobalNameSpace.FindLocal("i")
	assert.NotEquals(t, "i", i, NoValue)
	assert.Equals(t, "i.Type()", i.Type(), reflect.TypeOf(new(bool)).Elem())
	assert.Equals(t, "i", i.Interface(), false)
}

func TestFuncCall(t *testing.T) {
	mch := newMachine()
	assert.NoError(t, mch.Run(`const n = 500000000`))
	assert.NoError(t, mch.Run(`fmt.Println(math.Sin(n))`))
	
	assert.NoError(t, mch.Run(`i := fmt.Sprint(reflect.ValueOf(10).Type())`))
	i := mch.GlobalNameSpace.FindLocal("i")
	if assert.NotEquals(t, "i", i, NoValue) {
		assert.Equals(t, "i", i.Interface(), "int")
	}
}

func TestMake(t *testing.T) {
	mch := newMachine()
	assert.NoError(t, mch.Run(`s := make([]string, 3)
l := len(s)`))
	l := mch.GlobalNameSpace.FindLocal("l")
	if assert.NotEquals(t, "l", l, NoValue) {
		assert.Equals(t, "l", l.Interface(), 3)
	}
}