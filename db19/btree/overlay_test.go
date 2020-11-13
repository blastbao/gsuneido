// Copyright Suneido Software Corp. All rights reserved.
// Governed by the MIT license found in the LICENSE file.

package btree

import (
	"math/rand"
	"sort"
	"testing"

	"github.com/apmckinlay/gsuneido/db19/btree/inter"
	"github.com/apmckinlay/gsuneido/db19/ixspec"
	"github.com/apmckinlay/gsuneido/db19/stor"
	"github.com/apmckinlay/gsuneido/util/assert"
	"github.com/apmckinlay/gsuneido/util/str"
)

func TestEmptyOverlay(t *testing.T) {
	var data []string
	GetLeafKey = func(_ *stor.Stor, _ *ixspec.T, i uint64) string { return data[i] }
	defer func(mns int) { MaxNodeSize = mns }(MaxNodeSize)
	MaxNodeSize = 64
	fb := CreateFbtree(stor.HeapStor(8192), nil)
	mut := &inter.T{}
	u := &inter.T{}
	ov := &Overlay{fb: fb, under: []*inter.T{u}, mut: mut}
	checkIter(t, data, ov)

	const n = 100
	randKey := str.UniqueRandom(3, 8)

	data = insert(data, n, randKey, mut)
	checkIter(t, data, ov)

	data = insert(data, n, randKey, u)
	checkIter(t, data, ov)

	for i := 0; i < n/2; i++ {
		j := rand.Intn(len(data))
		if data[j] != "" {
			ov.Delete(data[j], key2off(data[j]))
			data[j] = ""
		}
	}
	checkIter(t, data, ov)
}

type insertable interface {
	Insert(key string, off uint64)
}

func insert(data []string, n int, randKey func() string, dest insertable) []string {
	for i := 0; i < n; i++ {
		key := randKey()
		off := key2off(key)
		data = append(data, key)
		dest.Insert(key, off)
	}
	return data
}

func key2off(key string) uint64 {
	off := uint64(0)
	for _, c := range key {
		off = off<<8 + uint64(c)
	}
	return off
}

func checkIter(t *testing.T, data []string, tr tree) {
	assert := assert.T(t)
	sort.Strings(data)
	it := tr.Iter(true)
	for _, k := range data {
		if k == "" {
			continue
		}
		k2, o2, ok := it()
		assert.True(ok)
		assert.This(k2).Is(k)
		assert.This(o2).Is(key2off(k))
	}
	_, _, ok := it()
	assert.False(ok)
}

func TestOverlayMerge(t *testing.T) {
	randKey := str.UniqueRandomOf(3, 10, "abcdef")
	var data []string
	randInter := func() *inter.T {
		const n = 300
		mut := &inter.T{}
		for i := 0; i < n; i++ {
			key := randKey()
			off := uint64(len(data))
			data = append(data, key)
			mut.Insert(key, off)
		}
		return mut
	}
	mut := randInter()
	GetLeafKey = func(_ *stor.Stor, _ *ixspec.T, i uint64) string { return data[i] }
	defer func(mns int) { MaxNodeSize = mns }(MaxNodeSize)
	MaxNodeSize = 64
	fb := CreateFbtree(stor.HeapStor(8192), nil)
	bi := &inter.T{}
	ov := Overlay{fb: fb, under: []*inter.T{bi, mut}}
	bi = ov.Merge(1)
	checkData(t, bi, data)

	mut = randInter()
	ov = Overlay{fb: fb, under: []*inter.T{bi, mut}}
	bi = ov.Merge(1)
	checkData(t, bi, data)
}

func checkData(t *testing.T, bi *inter.T, data []string) {
	t.Helper()
	assert.T(t).This(bi.Len()).Is(len(data))
	sort.Strings(data)
	i := 0
	n := 0
	it := bi.Iter(false)
	for key,_,ok := it(); ok; key,_,ok = it() {
		assert.T(t).This(key).Is(data[i])
		i++
		n++
	}
	assert.T(t).This(n).Is(len(data))
}
