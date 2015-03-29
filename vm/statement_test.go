package gsvm

import (
	"fmt"
	"testing"

	"github.com/daviddengcn/go-assert"
)

func TestForStatement(t *testing.T) {
	mch := newMachine()

	assert.NoError(t, mch.Run(`var sum, j int`))
	sum := mch.GlobalNameSpace.FindLocal("sum")
	assert.NotEquals(t, "sum", sum, NoValue)

	j := mch.GlobalNameSpace.FindLocal("j")
	assert.NotEquals(t, "j", j, NoValue)

	if assert.NoError(t, mch.Run(`sum = 0
j = 10
for i := 0; i < 10; i ++ {
	sum += i
	j--
}`)) {
		assert.Equals(t, "sum", sum.Interface(), 45)
		assert.Equals(t, "j", j.Interface(), 0)
	}

	if assert.NoError(t, mch.Run(`sum, j = 0, 10
for i := 0; i < 10; i ++ {
	sum += i
	j -= 1
	if i >= 5 {
		break
	} else {
		fmt.Println(i)
	}
}`)) {
		assert.Equals(t, "sum", sum.Interface(), 15)
		assert.Equals(t, "j", j.Interface(), 4)
	}
}

func TestOpAssignStatement(t *testing.T) {
	mch := newMachine()

	assert.NoError(t, mch.Run(`s := "abc"`))
	s := mch.GlobalNameSpace.FindLocal("s")
	assert.NotEquals(t, "s", s, NoValue)
	assert.Equals(t, "s", s.Interface(), "abc")

	assert.NoError(t, mch.Run(`s += "def"`))
}

func TestSwitchStatment(t *testing.T) {
	mch := newMachine()

	assert.NoError(t, mch.Run(`i, s := 2, ""`))
	assert.NoError(t, mch.Run(`switch i {
	case 2:
		s = "two"
}`))
	s := mch.GlobalNameSpace.FindLocal("s")
	assert.Equals(t, "s", s.Interface(), "two")

	// check execution of default clause
	assert.NoError(t, mch.Run(`j := 3
switch {
	case j == 2:
		j = 4
	default:
		j = 5
}`))
	j := mch.GlobalNameSpace.FindLocal("j")
	assert.Equals(t, "j", j.Interface(), 5)
}

func TestAppend(t *testing.T) {
	mch := newMachine()

	assert.NoError(t, mch.Run(`s := []string{"abc"}
s = append(s, "def")`))
	s := mch.GlobalNameSpace.FindLocal("s")
	assert.Equals(t, "s", fmt.Sprint(s.Interface()), fmt.Sprint([]string{"abc", "def"}))
}

func TestDelete(t *testing.T) {
	mch := newMachine()

	assert.NoError(t, mch.Run(`m := map[string]int{
	"abc": 1,
	"def": 2,
}
l := len(m)`))
	assert.Equals(t, "l", mch.GlobalNameSpace.FindLocal("l").Interface(), 2)

	assert.NoError(t, mch.Run(`delete(m, "ghg")
l = len(m)`))
	assert.Equals(t, "l", mch.GlobalNameSpace.FindLocal("l").Interface(), 2)

	assert.NoError(t, mch.Run(`delete(m, "abc")
l = len(m)`))
	assert.Equals(t, "l", mch.GlobalNameSpace.FindLocal("l").Interface(), 1)
}

func TestRange(t *testing.T) {
	mch := newMachine()

	assert.NoError(t, mch.Run(`nums := []int{2, 3, 4}
sum := 0
for _, num := range nums {
    sum += num
}`))
	assert.Equals(t, "sum", mch.GlobalNameSpace.FindLocal("sum").Interface(), 9)

	assert.NoError(t, mch.Run(`kvs := map[string]int{"a": 1, "b": 2}
sum = 0
for k, v := range kvs {
    fmt.Printf("%s -> %d\n", k, v)
	sum += v
}`))
	assert.Equals(t, "sum", mch.GlobalNameSpace.FindLocal("sum").Interface(), 3)

	assert.NoError(t, mch.Run(`sum = 0
for i, c := range "go" {
	sum += i + int(c)
}`))
	assert.Equals(t, "sum", mch.GlobalNameSpace.FindLocal("sum").Interface(), 215)
}
