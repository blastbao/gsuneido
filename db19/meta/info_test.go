// Copyright Suneido Software Corp. All rights reserved.
// Governed by the MIT license found in the LICENSE file.

package meta

import (
	"testing"

	"github.com/apmckinlay/gsuneido/db19/stor"
	"github.com/apmckinlay/gsuneido/util/assert"
	"github.com/apmckinlay/gsuneido/util/str"
)

func allInfo(*Info) bool { return true }

func TestInfo(t *testing.T) {
	assert := assert.T(t).This
	one := &Info{
		Table: "one",
		Nrows: 100,
		Size:  1000,
	}
	two := &Info{
		Table: "two",
		Nrows: 200,
		Size:  2000,
	}
	tbl := InfoHamt{}.Mutable()
	tbl.Put(one)
	tbl.Put(two)

	st := stor.HeapStor(8192)
	st.Alloc(1) // avoid offset 0
	off := tbl.Write(st, 0, allInfo)

	tbl, _ = ReadInfoChain(st, off)
	x, _ := tbl.Get("one")
	assert(*x).Is(*one)
	x, _ = tbl.Get("two")
	assert(*x).Is(*two)
}

func TestInfo2(t *testing.T) {
	tbl := InfoHamt{}.Mutable()
	const n = 1000
	data := mkdata(tbl, n)
	st := stor.HeapStor(32 * 1024)
	st.Alloc(1) // avoid offset 0
	off := tbl.Write(st, 0, allInfo)

	tbl, _ = ReadInfoChain(st, off)
	for i, s := range data {
		ti, _ := tbl.Get(s)
		assert.T(t).Msg("table").This(ti.Table).Is(s)
		assert.T(t).Msg("nrows").This(ti.Nrows).Is(i)
		_, ok := tbl.Get(s + "Z")
		assert.T(t).That(!ok)
	}
}

func mkdata(tbl InfoHamt, n int) []string {
	data := make([]string, n)
	randStr := str.UniqueRandom(4, 4)
	for i := 0; i < n; i++ {
		data[i] = randStr()
		tbl.Put(&Info{Table: data[i], Nrows: i})
	}
	return data
}
