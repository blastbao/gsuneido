// Copyright Suneido Software Corp. All rights reserved.
// Governed by the MIT license found in the LICENSE file.

package ixkey

import (
	"math/rand"
	"strings"
	"testing"

	. "github.com/apmckinlay/gsuneido/runtime"
	"github.com/apmckinlay/gsuneido/util/assert"
)

func TestEncoder(t *testing.T) {
	assert := assert.T(t).This
	enc := Encoder{}
	enc.Add("a")
	enc.Add("b")
	assert(enc.String()).Is("a\x00\x00b")
	enc.Add("a")
	enc.Add("b")
	enc.Add("c")
	assert(enc.String()).Is("a\x00\x00b\x00\x00c")
	enc.Add("a\x00b")
	enc.Add("c")
	assert(enc.String()).Is("a\x00\x01b\x00\x00c")
}

func TestKey(t *testing.T) {
	assert := assert.T(t).This

	// no escape for single field
	assert(key(mkrec("a\x00b"), []int{0}, nil)).Is("a\x00b")

	fields := []int{0, 1, 2}
	for _, flds2 := range [][]int{nil, {1, 2}} {
		assert(key(mkrec("a", "b"), []int{}, flds2)).Is("")
		assert(key(mkrec("a", "b"), []int{0}, flds2)).Is("a")
		assert(key(mkrec("a", "b"), []int{1}, flds2)).Is("b")
		assert(key(mkrec("a", "b"), []int{0, 1}, flds2)).Is("a\x00\x00b")
		assert(key(mkrec("a", "b"), []int{1, 0}, flds2)).Is("b\x00\x00a")

		// omit trailing empty fields
		assert(key(mkrec("a", "b", "c"), fields, flds2)).Is("a\x00\x00b\x00\x00c")
		assert(key(mkrec("a", "", "c"), fields, flds2)).Is("a\x00\x00\x00\x00c")
		assert(key(mkrec("", "", "c"), fields, flds2)).Is("\x00\x00\x00\x00c")
		assert(key(mkrec("a", "b", ""), fields, flds2)).Is("a\x00\x00b")
		assert(key(mkrec("a", "", ""), fields, flds2)).Is("a")

		// escaping
		first := []int{0, 1}
		assert(key(mkrec("ab"), first, flds2)).Is("ab")
		assert(key(mkrec("a\x00b"), first, flds2)).Is("a\x00\x01b")
		assert(key(mkrec("\x00ab"), first, flds2)).Is("\x00\x01ab")
		assert(key(mkrec("a\x00\x00b"), first, flds2)).Is("a\x00\x01\x00\x01b")
		assert(key(mkrec("a\x00\x01b"), first, flds2)).Is("a\x00\x01\x01b")
		assert(key(mkrec("ab\x00"), first, flds2)).Is("ab\x00\x01")
		assert(key(mkrec("ab\x00\x00"), first, flds2)).Is("ab\x00\x01\x00\x01")
	}

	// fields2
	fields2 := []int{3, 4}
	assert(key(mkrec("", "", ""), fields, nil)).Is("")
	assert(key(mkrec("", "", "", "a", "b"), fields, fields2)).
		Is("\x00\x00\x00\x00\x00\x00a\x00\x00b")
	assert(key(mkrec("x", "", "", "a", "b"), fields, fields2)).
		Is("x")
}

func key(rec Record, flds, flds2 []int) string {
	spec := Spec{Fields: flds, Fields2: flds2}
	return spec.Key(rec)
}

func mkrec(args ...string) Record {
	var b RecordBuilder
	for _, a := range args {
		b.AddRaw(a)
	}
	return b.Build()
}

const m = 3

func TestKeyBug(t *testing.T) {
	fields := []int{0}
	fields2 := []int{1}
	k1 := key(mkrec("", "foo"), fields, fields2)
	k2 := key(mkrec("\x00\x00foo"), fields, fields2)
	assert.T(t).That(k1 != k2)
}

func TestRandom(t *testing.T) {
	assert := assert.T(t).This
	var n = 100000
	if testing.Short() {
		n = 10000
	}
	fields := []int{0, 1, 2}
	for i := 0; i < n; i++ {
		x := gen()
		y := gen()
		yenc := key(y, fields, nil)
		xenc := key(x, fields, nil)
		assert(xenc < yenc).Is(lt(x, y))
		assert(strings.Compare(xenc, yenc)).Is(compare(x, y, fields, nil))
	}
}

func compare(r1, r2 Record, flds, flds2 []int) int {
	spec := Spec{Fields: flds, Fields2: flds2}
	return spec.Compare(r1, r2)
}

func gen() Record {
	var b RecordBuilder
	for i := 0; i < m; i++ {
		x := make([]byte, rand.Intn(6)+1)
		for j := range x {
			x[j] = byte(rand.Intn(4)) // 25% zeros
		}
		b.AddRaw(string(x))
	}
	return b.Build()
}

func lt(x Record, y Record) bool {
	for i := 0; i < x.Len() && i < y.Len(); i++ {
		if cmp := strings.Compare(x.GetRaw(i), y.GetRaw(i)); cmp != 0 {
			return cmp < 0
		}
	}
	return x.Len() < y.Len()
}
