package gsvm

import (
	"fmt"
	"math"
	"reflect"
	"testing"

	"github.com/daviddengcn/go-assert"
)

func newMachine() *machine {
	return New(&PackageNameSpace{Packages: map[string]Package{
		"fmt": Package{
			"Println": reflect.ValueOf(fmt.Println),
			"Sprint":  reflect.ValueOf(fmt.Sprint),
			"Printf":  reflect.ValueOf(fmt.Printf),
		},
		"math": Package{
			"Sin": reflect.ValueOf(math.Sin),
		},
		"reflect": Package{
			"ValueOf": reflect.ValueOf(reflect.ValueOf),
			"TypeOf":  reflect.ValueOf(reflect.TypeOf),
		},
		"gsvm": Package{
			"TypeValue": reflect.ValueOf(TypeValue{reflect.TypeOf(TypeValue{})}),
		},
	}}).(*machine)
}

func TestTypeLiteralConvert(t *testing.T) {
	mch := newMachine()

	if assert.NoError(t, mch.Run(`var j = 1.5`)) {
		j := mch.GlobalNameSpace.FindLocal("j")
		if assert.NotEquals(t, "j", j, NoValue) {
			assert.Equals(t, "j.Type()", j.Type(), reflect.TypeOf(new(float64)).Elem())
			assert.Equals(t, "j", j.Interface(), 1.5)
		}
	}

	if assert.NoError(t, mch.Run(`var k float32`)) {
		k := mch.GlobalNameSpace.FindLocal("k")
		if assert.NotEquals(t, "k", k, NoValue) {
			assert.Equals(t, "k.Type()", k.Type(), reflect.TypeOf(new(float32)).Elem())
			assert.Equals(t, "k", k.Interface(), float32(0))
		}
	}

	if assert.NoError(t, mch.Run(`var l complex128 = 1`)) {
		l := mch.GlobalNameSpace.FindLocal("l")
		if assert.NotEquals(t, "l", l, NoValue) {
			assert.Equals(t, "l.Type()", l.Type(), reflect.TypeOf(new(complex128)).Elem())
			assert.Equals(t, "l", l.Interface(), complex(1, 0))
		}
	}

	if assert.NoError(t, mch.Run(`i, k := 1, 2`)) {
		i := mch.GlobalNameSpace.FindLocal("i")
		if assert.NotEquals(t, "i", i, NoValue) {
			assert.Equals(t, "i.Type()", i.Type(), reflect.TypeOf(new(int)).Elem())
			assert.Equals(t, "i", i.Interface(), 1)
		}
		k := mch.GlobalNameSpace.FindLocal("k")
		if assert.NotEquals(t, "k", k, NoValue) {
			assert.Equals(t, "k.Type()", k.Type(), reflect.TypeOf(new(float32)).Elem())
			assert.Equals(t, "k", k.Interface(), float32(2))
		}
	}

	if assert.NoError(t, mch.Run(`l = 3`)) {
		l := mch.GlobalNameSpace.FindLocal("l")
		if assert.NotEquals(t, "l", l, NoValue) {
			assert.Equals(t, "l.Type()", l.Type(), reflect.TypeOf(new(complex128)).Elem())
			assert.Equals(t, "l", l.Interface(), complex(3, 0))
		}
	}
}

func TestConstantType(t *testing.T) {
	mch := newMachine()

	assert.NoError(t, mch.Run(`const n = 500000000`))
	assert.NoError(t, mch.Run(`const d = 3e20 / n`))
}

func TestTypeConversion(t *testing.T) {
	mch := newMachine()
	assert.NoError(t, mch.Run(`i := 10`))
	assert.NoError(t, mch.Run(`j := int64(i)`))

	i := mch.GlobalNameSpace.FindLocal("i")
	if assert.NotEquals(t, "i", i, NoValue) {
		assert.Equals(t, "i.Type()", i.Type(), reflect.TypeOf(new(int)).Elem())
	}
	j := mch.GlobalNameSpace.FindLocal("j")
	if assert.NotEquals(t, "j", j, NoValue) {
		assert.Equals(t, "j.Type()", j.Type(), reflect.TypeOf(new(int64)).Elem())
	}
}

func TestArrayType(t *testing.T) {
	//	mch := newMachine()
	//	assert.NoError(t, mch.Run(`var a [5]int`))
}

func TestSliceType(t *testing.T) {
	mch := newMachine()
	assert.NoError(t, mch.Run(`s := make([]string, 3)
s[0] = "abc"
e := s[0]`))
	e := mch.GlobalNameSpace.FindLocal("e")
	if assert.NotEquals(t, "e", e, NoValue) {
		assert.Equals(t, "e.Type()", e.Type(), reflect.TypeOf(new(string)).Elem())
		assert.Equals(t, "e", e.Interface(), "abc")
	}
}

func TestMapType(t *testing.T) {
	mch := newMachine()

	assert.NoError(t, mch.Run(`m := make(map[string]int)
m["k1"] = 7
v := m["k1"]`))
	assert.StringEquals(t, "v", mch.GlobalNameSpace.FindLocal("v").Interface(), 7)
}
