package gsvm

import (
	"testing"

	"github.com/daviddengcn/go-assert"
)

func TestForStatement(t *testing.T) {
	mch := newMachine()

	assert.NoError(t, mch.Run(`var sum, j int`))
	psum := mch.GlobalNameSpace.FindLocalVar("sum")
	assert.NotEquals(t, "psum", psum, noValue)

	pj := mch.GlobalNameSpace.FindLocalVar("j")
	assert.NotEquals(t, "pj", pj, noValue)

	if assert.NoError(t, mch.Run(`sum = 0
j = 10
for i := 0; i < 10; i ++ {
	sum += i
	j--
}`)) {
		assert.Equals(t, "sum", psum.Elem().Interface(), 45)
		assert.Equals(t, "j", pj.Elem().Interface(), 0)
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
		assert.Equals(t, "sum", psum.Elem().Interface(), 15)
		assert.Equals(t, "j", pj.Elem().Interface(), 4)
	}
}

func TestOpAssignStatement(t *testing.T) {
	mch := newMachine()

	assert.NoError(t, mch.Run(`s := "abc"`))
	ps := mch.GlobalNameSpace.FindLocalVar("s")
	assert.NotEquals(t, "ps", ps, noValue)
	assert.Equals(t, "ps", ps.Elem().Interface(), "abc")

	assert.NoError(t, mch.Run(`s += "def"`))
}
