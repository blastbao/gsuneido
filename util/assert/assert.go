// Copyright Suneido Software Corp. All rights reserved.
// Governed by the MIT license found in the LICENSE file.

// Package assert helps writing assertions for tests.
// Benefits include brevity, clarity, and helpful error messages.
//
// If .T(t) is specified, failures are reported with t.Error
// which means multiple errors may be reported.
// If .T(t) is not specified, panic is called with the error string.
//
// For example:
//		assert.That(a || b)
//		assert.This(x).Is(y)
//		assert.T(t).This(x).Like(y)
//		assert.Msg("first time").That(a || b)
//		assert.T(t).Msg("second").This(fn).Panics("illegal")
//
// Use a variable to avoid specifying .T(t) repeatedly:
//		assert := assert.T(t)
//		assert.This(x).Is(y)
package assert

import (
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

type assert struct {
	t   *testing.T
	msg []interface{}
}

// T specifies a *testing.T to use for reporting errors
func T(t *testing.T) assert {
	return assert{t: t}
}

// Msg adds additional information to print with the error.
// It can be useful with That/True/False where the error is not very helpful.
func Msg(args ...interface{}) assert {
	return assert{msg: args}
}

// Msg adds additional information to print with the error.
// It can be useful with That/True/False where the error is not very helpful.
func (a assert) Msg(args ...interface{}) assert {
	a.msg = args
	return a
}

// Msg adds additional information to print with the error.
// It can be useful with That/True/False where the error is not very helpful.
func (v value) Msg(args ...interface{}) value {
	v.assert.msg = args
	return v
}

// Nil gives an error if the value is not nil.
// It handles nil func/pointer/slice/map/channel using reflect.IsNil
// For performance critical code, consider using That(value == nil)
func Nil(v interface{}) {
	assert{}.This(v).Is(nil)
}

// Nil gives an error if the value is not nil.
// It handles nil func/pointer/slice/map/channel using reflect.IsNil
// For performance critical code, consider using That(value == nil)
func (a assert) Nil(v interface{}) {
	if a.t != nil {
		a.t.Helper()
	}
	a.This(v).Is(nil)
}

// True gives an error if the value is not true.
// True(x) is the same as That(x)
func True(b bool) {
	assert{}.That(b)
}

// True gives an error if the value is not true.
// True(x) is the same as That(x)
func (a assert) True(b bool) {
	if a.t != nil {
		a.t.Helper()
	}
	a.That(b)
}

// False gives an error if the value is not true.
// False(x) is the same as That(!x)
func False(b bool) {
	assert{}.That(!b)
}

// False gives an error if the value is not true.
// False(x) is the same as That(!x)
func (a assert) False(b bool) {
	if a.t != nil {
		a.t.Helper()
	}
	a.That(!b)
}

// That gives an error if the value is not true.
// That(x) is the same as True(x)
func That(cond bool) {
	if !cond { // redundant inline for speed
		assert{}.That(cond)
	}
}

// That gives an error if the value is not true.
// That(x) is the same as True(x)
func (a assert) That(cond bool) {
	if !cond {
		if a.t != nil {
			a.t.Helper()
		}
		a.fail("assert failed")
	}
}

// This sets a value to be compared e.g. with Is or Like
func This(v interface{}) value {
	return value{value: v}
}

// This sets a value to be compared e.g. with Is or Like.
// It is usually the actual value and Is gives the expected.
func (a assert) This(v interface{}) value {
	return value{assert: a, value: v}
}

type value struct {
	assert assert
	value  interface{}
}

// Is gives an error if the given expected value is not the same
// as the actual value supplied to This.
// Accepts as equivalent: different nils and different int types.
// Compares floats via string forms.
// Uses an Equal method if available on the expected value.
// Finally, uses reflect.DeepEqual.
func (v value) Is(expected interface{}) {
	if !Is(v.value, expected) {
		if v.assert.t != nil {
			v.assert.t.Helper()
		}
		v.assert.fail("expected: ", show(expected),
			"\nactual: ", show(v.value))
	}
}

// Isnt gives an error if the given expected value is the same
// as the actual value supplied to This.
func (v value) Isnt(expected interface{}) {
	if Is(v.value, expected) {
		if v.assert.t != nil {
			v.assert.t.Helper()
		}
		v.assert.fail("expected not: ", show(expected), " but it was")
	}
}

func Is(actual, expected interface{}) bool {
	if isNil(expected) && isNil(actual) {
		return true
	}
	if a, ok := actual.(float64); ok {
		if e, ok := expected.(float64); ok {
			if strconv.FormatFloat(a, 'e', 15, 64) ==
				strconv.FormatFloat(e, 'e', 15, 64) {
				return true
			}
		}
	}
	if intEqual(expected, actual) {
		return true
	}
	type equal interface {
		Equal(interface{}) bool
	}
	if e, ok := expected.(equal); ok {
		if e.Equal(actual) {
			return true
		}
	} else if reflect.DeepEqual(expected, actual) {
		return true
	}
	return false
}

func isNil(x interface{}) bool {
	if x == nil {
		return true
	}
	v := reflect.ValueOf(x)
	switch v.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return v.IsNil()
	}
	return false
}

func intEqual(x interface{}, y interface{}) bool {
	var x64 int64
	switch x := x.(type) {
	case int:
		x64 = int64(x)
	case uint:
		x64 = int64(x)
	case int8:
		x64 = int64(x)
	case uint8:
		x64 = int64(x)
	case int16:
		x64 = int64(x)
	case uint16:
		x64 = int64(x)
	case int32:
		x64 = int64(x)
	case uint32:
		x64 = int64(x)
	case int64:
		x64 = int64(x)
	case uint64:
		x64 = int64(x)
	default:
		return false
	}
	switch y := y.(type) {
	case int:
		return x64 == int64(y)
	case uint:
		return x64 == int64(y)
	case int8:
		return x64 == int64(y)
	case uint8:
		return x64 == int64(y)
	case int16:
		return x64 == int64(y)
	case uint16:
		return x64 == int64(y)
	case int32:
		return x64 == int64(y)
	case uint32:
		return x64 == int64(y)
	case int64:
		return x64 == int64(y)
	case uint64:
		return x64 == int64(y)
	default:
		return false
	}
}

func show(x interface{}) string {
	if _, ok := x.(string); ok {
		return fmt.Sprintf("%#v", x)
	}
	s1 := fmt.Sprintf("%v", x)
	s2 := fmt.Sprintf("%#v", x)
	if s1[0] == '[' {
		return s2
	}
	if s1 == s2 {
		return s1 + " (" + fmt.Sprintf("%T", x) + ")"
	}
	return s1 + " (" + s2 + ")"
}

// Like compares strings with whitespace standardized.
// Leading and trailing whitespace is removed,
// runs of whitespace are converted to a single space.
func (v value) Like(expected interface{}) {
	exp := expected.(string)
	val := v.value.(string)
	if !like(exp, val) {
		if v.assert.t != nil {
			v.assert.t.Helper()
		}
		sep := " "
		if strings.Contains(exp, "\n") || strings.Contains(val, "\n") {
			sep = "\n"
		}
		v.assert.fail("expected:" + sep + exp + "\nbut got:" + sep + val)
	}
}

func like(expected, actual string) bool {
	return canon(actual) == canon(expected)
}

func canon(s string) string {
	s = strings.TrimSpace(s)
	s = leadingSpace.ReplaceAllString(s, "")
	s = trailingSpace.ReplaceAllString(s, "")
	s = whitespace.ReplaceAllString(s, " ")
	return s
}

var leadingSpace = regexp.MustCompile("(?m)^[ \t]+")
var trailingSpace = regexp.MustCompile("(?m)[ \t]+$")
var whitespace = regexp.MustCompile("[ \t]+")

// Panics checks if a function panics
func (v value) Panics(expected string) {
	e := Catch(v.value.(func()))
	if e == nil {
		if v.assert.t != nil {
			v.assert.t.Helper()
		}
		v.assert.fail(fmt.Sprintf("expected panic with '%v' but it did not panic",
			expected))
		return
	}
	if err, ok := e.(error); ok {
		e = err.Error()
	}
	if !strings.Contains(e.(string), expected) {
		if v.assert.t != nil {
			v.assert.t.Helper()
		}
		v.assert.fail(fmt.Sprintf("expected panic with '%v' but got '%v'",
			expected, e))
	}
}

// Catch calls the given function, catching and returning panics
func Catch(f func()) (result interface{}) {
	defer func() {
		if e := recover(); e != nil {
			//debug.PrintStack()
			result = e
		}
	}()
	f()
	return
}

//-------------------------------------------------------------------

func (a assert) fail(args ...interface{}) {
	// fmt.Println("==============================")
	// debug.PrintStack()
	if a.t != nil {
		a.t.Helper()
	}
	if len(a.msg) > 0 {
		args = append(append(args, "\nmsg: "), a.msg...)
	}
	s := fmt.Sprintln(args...)
	s = strings.TrimRight(s, "\r\n")
	if a.t != nil {
		a.t.Error("\n" + s)
	} else {
		panic("assert failed: " + getLocation() + "\n" + s)
	}
}

func getLocation() string {
	i := 1
	for ; i < 9; i++ {
		_, file, _, ok := runtime.Caller(i)
		if !ok || strings.Contains(file, "testing/testing.go") {
			break
		}
	}
	_, file, line, ok := runtime.Caller(i - 1)
	if !ok || i <= 1 || i >= 9 {
		return ""
	}
	// Truncate file name at last separator.
	if index := strings.LastIndex(file, "/"); index >= 0 {
		file = file[index+1:]
	} else if index = strings.LastIndex(file, "\\"); index >= 0 {
		file = file[index+1:]
	}
	return file + ":" + strconv.Itoa(line)
}
