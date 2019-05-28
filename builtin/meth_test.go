package builtin

import (
	"testing"

	. "github.com/apmckinlay/gsuneido/runtime"
	. "github.com/apmckinlay/gsuneido/util/hamcrest"
)

func Test_MethodCall(t *testing.T) {
	n := NumFromString("12.34")
	th := NewThread()
	f := n.Lookup(th, "Round")
	th.Push(IntVal(1))
	result := f.Call(th, n, ArgSpec1)
	Assert(t).That(result, Equals(NumFromString("12.3")))
}
