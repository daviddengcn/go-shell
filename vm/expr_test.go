package gsvm

import (
	"fmt"
	"image/color"
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

func TestCompositeLit(t *testing.T) {
	mch := newMachine()

	assert.NoError(t, mch.Run(`s := []string{"abc"}
l := len(s)
str := fmt.Sprint(s)`))
	l := mch.GlobalNameSpace.FindLocal("l")
	if assert.NotEquals(t, "l", l, NoValue) {
		assert.Equals(t, "l", l.Interface(), 1)
	}
	str := mch.GlobalNameSpace.FindLocal("str")
	if assert.NotEquals(t, "str", str, NoValue) {
		assert.Equals(t, "str", str.Interface(), fmt.Sprint([]string{"abc"}))
	}

	assert.NoError(t, mch.Run(`a := color.Alpha{A: 10}
aa := a.A`))
	assert.Equals(t, "a", mch.GlobalNameSpace.FindLocal("a").Interface(), color.Alpha{A: 10})
	assert.Equals(t, "aa", mch.GlobalNameSpace.FindLocal("aa").Interface(), uint8(10))

	assert.NoError(t, mch.Run(`b := color.Alpha{20}
pb := &b
ba := pb.A`))
	assert.Equals(t, "b", mch.GlobalNameSpace.FindLocal("b").Interface(), color.Alpha{20})
	assert.Equals(t, "aa", mch.GlobalNameSpace.FindLocal("ba").Interface(), uint8(20))
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

func TestCopy(t *testing.T) {
	mch := newMachine()
	assert.NoError(t, mch.Run(`s := []string{"abc", "def"}
t := []string{""}
l := copy(t, s)`))
	assert.Equals(t, "l", mch.GlobalNameSpace.FindLocal("l").Interface(), 1)
	assert.StringEquals(t, "t", mch.GlobalNameSpace.FindLocal("t").Interface(), []string{"abc"})
}

func TestSlicing(t *testing.T) {
	mch := newMachine()
	assert.NoError(t, mch.Run(`s := []string{"abc", "def", "ghi", "j", "k"}
l := s[2:5]
m := s[:3]
k := s[1:2:3]`))
	l := mch.GlobalNameSpace.FindLocal("l").Interface().([]string)
	assert.StringEquals(t, "l", l, []string{"ghi", "j", "k"})
	assert.Equals(t, "cap(l)", cap(l), 3)

	m := mch.GlobalNameSpace.FindLocal("m").Interface().([]string)
	assert.StringEquals(t, "m", m, []string{"abc", "def", "ghi"})
	assert.Equals(t, "cap(m)", cap(m), 5)

	k := mch.GlobalNameSpace.FindLocal("k").Interface().([]string)
	assert.StringEquals(t, "k", k, []string{"def"})
	assert.Equals(t, "cap(k)", cap(k), 2)
}

func TestMap(t *testing.T) {
	mch := newMachine()

	assert.NoError(t, mch.Run(`m := map[string]int{
	"k1": 7,
	"k2": 13,
}
k := m["k1"]`))
	assert.Equals(t, "k", mch.GlobalNameSpace.FindLocal("k").Interface(), 7)

	assert.NoError(t, mch.Run(`l, ok := m["k2"]`))
	assert.Equals(t, "l", mch.GlobalNameSpace.FindLocal("l").Interface(), 13)
	assert.Equals(t, "ok", mch.GlobalNameSpace.FindLocal("ok").Interface(), true)

	assert.NoError(t, mch.Run(`l, ok = m["k3"]`))
	assert.Equals(t, "l", mch.GlobalNameSpace.FindLocal("l").Interface(), 0)
	assert.Equals(t, "ok", mch.GlobalNameSpace.FindLocal("ok").Interface(), false)

	assert.NoError(t, mch.Run(`l = m["k1"]`))
	assert.Equals(t, "l", mch.GlobalNameSpace.FindLocal("l").Interface(), 7)
}

func TestClosure(t *testing.T) {
	/* wait for reflect to support FuncOf
		mch := newMachine()
		assert.NoError(t, mch.Run(`
	var nextInt func() int
	{
		i := 0
	}`))
	*/
}
