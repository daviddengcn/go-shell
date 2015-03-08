package gsvm

import (
	"reflect"
	"testing"

	"github.com/daviddengcn/go-assert"
)

func newMachine() *machine {
	return New().(*machine)
}

func TestTypeLiteralConvert(t *testing.T) {
	mch := newMachine()

	if assert.NoError(t, mch.Run(`var j = 1.5`)) {
		j, _ := mch.GlobalNameSpace.FindLocalVar("j")
		if assert.NotEquals(t, "j", j, noValue) {
			assert.Equals(t, "j.Type()", j.Type(), reflect.TypeOf(new(float64)))
			assert.Equals(t, "j", j.Elem().Interface(), 1.5)
		}
	}

	if assert.NoError(t, mch.Run(`var k float32`)) {
		k, _ := mch.GlobalNameSpace.FindLocalVar("k")
		if assert.NotEquals(t, "k", k, noValue) {
			assert.Equals(t, "k.Type()", k.Type(), reflect.TypeOf(new(float32)))
			assert.Equals(t, "k", k.Elem().Interface(), float32(0))
		}
	}

	if assert.NoError(t, mch.Run(`var l complex128 = 1`)) {
		l, _ := mch.GlobalNameSpace.FindLocalVar("l")
		if assert.NotEquals(t, "l", l, noValue) {
			assert.Equals(t, "l.Type()", l.Type(), reflect.TypeOf(new(complex128)))
			assert.Equals(t, "l", l.Elem().Interface(), complex(1, 0))
		}
	}

	if assert.NoError(t, mch.Run(`i, k := 1, 2`)) {
		i, _ := mch.GlobalNameSpace.FindLocalVar("i")
		if assert.NotEquals(t, "i", i, noValue) {
			assert.Equals(t, "i.Type()", i.Type(), reflect.TypeOf(new(int)))
			assert.Equals(t, "i", i.Elem().Interface(), 1)
		}
		k, _ := mch.GlobalNameSpace.FindLocalVar("k")
		if assert.NotEquals(t, "k", k, noValue) {
			assert.Equals(t, "k.Type()", k.Type(), reflect.TypeOf(new(float32)))
			assert.Equals(t, "k", k.Elem().Interface(), float32(2))
		}
	}

	if assert.NoError(t, mch.Run(`l = 3`)) {
		l, _ := mch.GlobalNameSpace.FindLocalVar("l")
		if assert.NotEquals(t, "l", l, noValue) {
			assert.Equals(t, "l.Type()", l.Type(), reflect.TypeOf(new(complex128)))
			assert.Equals(t, "l", l.Elem().Interface(), complex(3, 0))
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

	i, _ := mch.GlobalNameSpace.FindLocalVar("i")
	if assert.NotEquals(t, "i", i, noValue) {
		assert.Equals(t, "i.Type()", i.Type(), reflect.TypeOf(new(int)))
	}
	j, _ := mch.GlobalNameSpace.FindLocalVar("j")
	if assert.NotEquals(t, "j", j, noValue) {
		assert.Equals(t, "j.Type()", j.Type(), reflect.TypeOf(new(int64)))
	}
}
