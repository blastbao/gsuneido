// Copyright Suneido Software Corp. All rights reserved.
// Governed by the MIT license found in the LICENSE file.

package compile

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/apmckinlay/gsuneido/runtime"
	"github.com/apmckinlay/gsuneido/util/assert"
	"github.com/apmckinlay/gsuneido/util/str"
)

func Example_GoGen1() {
	src := GoGen(`function (a, b) { a + b }`)
	fmt.Println(src)
	// output:
	//
	// func(a, b Value) Value {
	// return OpAdd(a, b)
	// }
}

func Example_GoGen2() {
	src := GoGen(`function (n)
		{
		sum = 0
		while (n > 0)
			{
			sum = sum + n
			n = n - 1
			}
		return sum
		}`)
	fmt.Println(src)
	// output:
	//
	// func(n Value) Value {
	// (sum := Zero)
	// for (n.Compare(Zero) > 0) {
	// (sum = OpAdd(sum, n))
	// (n = OpAdd(n, MinusOne))
	// }
	// return sum
	// }
}

func Example_GoGen3() {
	src := GoGen(`function ()
		{
		s = "hello"
		s + 123
		}`)
	fmt.Println(src)
	fmt.Println("_c0_", Unpack64(`BGhlbGxv`))
	fmt.Println("_c1_", Unpack64(`A4MMHg==`))
	// output:
	//
	// var _c0_ = Unpack64(`BGhlbGxv`)
	// var _c1_ = Unpack64(`A4MMHg==`)
	// func() Value {
	// (s := _c0_)
	// return OpAdd(s, _c1_)
	// }
	// _c0_ "hello"
	// _c1_ 123
}

func TestPack64(t *testing.T) {
	s := SuStr(strings.Repeat("hello world", 100))
	var g ggen
	g.pack64(s)
	b := g.init.String()
	b = str.AfterFirst(b, "`")
	b = str.BeforeLast(b, "`")
	assert.T(t).This(Unpack64(b)).Is(s)
}

func TestGoGen(t *testing.T) {
	stop := "-nothing-"
	test := func(src, expected string) {
		t.Helper()
		src = "function(a,b,c,d) {\n" + src + "\n}"
		code := GoGen(src)
		code = str.AfterFirst(code, "func(a, b, c, d Value) Value {\n")
		code = str.BeforeLast(code, "}")
		code = str.BeforeLast(code, stop)
		assert.T(t).This(code).Like(expected)
	}
	test("123;;", "return nil")
	test("", "return nil")
	test("return", "return nil")
	test("return false", "return False")
	stop = "\nreturn nil"
	test("a", "return a")
	test("a is b", "return SuBool((a.Equal(b)))")
	test("a isnt b;;", "(a.Equal(b) != true)")
	test("a > b", "return SuBool((a.Compare(b) > 0))")
	test("a <= b;;", "(a.Compare(b) <= 0)")
	test("+a;;", "OpUnaryPlus(a)")
	test("-a", "return OpUnaryMinus(a)")
	test("not a", "return OpNot(a)")
	test("a + b + c", "return OpAdd(OpAdd(a, b), c)")
	test("a + b - c", "return OpSub(OpAdd(a, b), c)")
	test("a - b + c", "return OpAdd(OpSub(a, b), c)")
	test("a and b and c;;", "(OpBool(a) && OpBool(b) && OpBool(c))")
	test("a or b or c", "return SuBool(OpBool(a) || OpBool(b) || OpBool(c))")
	test("a / b * c / d;;", "OpDiv(OpMul(a, c), OpMul(b, d))")
	test("1 / a", "return OpDiv(One, a)")
	test("a =~ b", "return OpMatch(t, a, b)")
	test("a << b;;", "OpLShift(a, b)")

	test("a = 1;;", "(a = One)")
	test("z = 1;;", "(z := One)")
	test("a = b", "return func(){ _r_ := b; (a = _r_); return _r_ }()")
	test("a[b] = 0;;", "a.Put(b, Zero)")
	test("a[b] = 0",
		"return func(){ _r_ := Zero; a.Put(b, _r_); return _r_ }()")
	test("a = a + b;;", "(a = OpAdd(a, b))")
	test("a += b;;", "a = OpAdd(a, b)")
	test("a = a + b",
		"return func(){ _r_ := OpAdd(a, b); (a = _r_); return _r_ }()")
	test("a += b",
		"return func(){ _r_ := OpAdd(a, b); a = _r_; return _r_ }()")
	test("a[b] += 1", "return a.GetPut(t, b, One, OpAdd, false)")
	test("++x;;", "x = OpAdd(x, One)")
	test("--x", "return func(){ _r_ := OpSub(x, One); x = _r_; return _r_ }()")
	test("x--;;", "x = OpSub(x, One)")
	test("x++", "return func(){ _r_ := x; x = OpAdd(_r_, One); return _r_ }()")
	test("a[b]++;;", "a.GetPut(t, b, One, OpAdd, false)")
	test("a[b]--", "return a.GetPut(t, b, One, OpSub, true)")
	test("--a[b];;", "a.GetPut(t, b, One, OpSub, false)")
	test("++a[b]", "return a.GetPut(t, b, One, OpAdd, false)")

	test("a ? b : c;;", "if OpBool(a) { b } else { c }")
	test("a ? b : c",
		"return func() { if OpBool(a) { return b } else { return c } }()")

	test("function(a,b){ a+b }", `return func(a, b Value) Value {
        return OpAdd(a, b)
		}`)
	test("forever { b; break }", `for {
		b
		break
		}`)
	test("while (a) { b; continue }", `for OpBool(a) {
		b
		continue
		}`)
	test("do { b } while (a)", `for {
		b
		if !OpBool(a) { break }
		}`)
	test("if (a) b", `if OpBool(a) {
        b
        }`)
	test("if (not a) b", `if !OpBool(a) {
        b
        }`)
	test("if (a) b else c", `if OpBool(a) {
        b
        } else {
		c
		}`)
	test("for (;;) a", `for ; ; {
        a
        }`)
	test("for (a,b; c; a) d", `a
		b
		for ; OpBool(c); a {
        d
        }`)
	test("for (; a; b,c) d", `for ; OpBool(a); func(){ b; c }() {
        d
		}`)
	test("for x in a { b }", `var x Value
        for _it_ := OpIter(a); ; {
        x = _it_.Next()
        if x == nil { break }
        b
		}`)
	test("throw a", "panic(a)")
	test("try a catch b", `func() {
		defer func() {
			if _e_ := recover(); _e_ != nil {
				OpCatch(t, _e_, "")
				b
			}
		}()
		a
		}()`)
	test("try a catch(x, 'uninit') b", `func() {
		defer func() {
			if _e_ := recover(); _e_ != nil {
				x = OpCatch(t, _e_, "uninit")
				b
			}
		}()
		a
		}()`)
	test("'hello'", "return _c0_")
}
