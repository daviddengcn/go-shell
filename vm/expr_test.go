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
	i, _ := mch.GlobalNameSpace.FindLocalVar("i")
	assert.NotEquals(t, "i", i, noValue)
	assert.Equals(t, "i.Type()", i.Type(), reflect.TypeOf(new(bool)))
	assert.Equals(t, "i", i.Elem().Interface(), false)
}

func TestFuncCall(t *testing.T) {
	mch := newMachine()
	assert.NoError(t, mch.Run(`const n = 500000000`))
	assert.NoError(t, mch.Run(`fmt.Println(math.Sin(n))`))
}
