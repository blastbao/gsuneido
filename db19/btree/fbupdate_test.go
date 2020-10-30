// Copyright Suneido Software Corp. All rights reserved.
// Governed by the MIT license found in the LICENSE file.

package btree

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/apmckinlay/gsuneido/db19/ixspec"
	"github.com/apmckinlay/gsuneido/db19/stor"
	"github.com/apmckinlay/gsuneido/runtime"
	"github.com/apmckinlay/gsuneido/util/assert"
	"github.com/apmckinlay/gsuneido/util/str"
)

func TestUpdate(t *testing.T) {
	var nTimes = 10
	if testing.Short() {
		nTimes = 1
	}
	for j := 0; j < nTimes; j++ {
		const n = 1000
		var data [n]string
		GetLeafKey = func(_ *stor.Stor, _ *ixspec.T, i uint64) string { return data[i] }
		defer func(mns int) { MaxNodeSize = mns }(MaxNodeSize)
		MaxNodeSize = 44
		fb := CreateFbtree(nil, nil)
		mfb := fb.makeMutable()
		randKey := str.UniqueRandomOf(3, 6, "abcde")
		for i := 0; i < n; i++ {
			key := randKey()
			data[i] = key
			mfb.Insert(key, uint64(i))
		}
		mfb.checkData(t, data[:])
		mfb.ckpaths()
	}
}

func TestUnevenSplit(t *testing.T) {
	const n = 1000
	var data [n]string
	test := func() {
		GetLeafKey = func(_ *stor.Stor, _ *ixspec.T, i uint64) string { return data[i] }
		defer func(mns int) { MaxNodeSize = mns }(MaxNodeSize)
		MaxNodeSize = 128
		fb := CreateFbtree(nil, nil)
		mfb := fb.makeMutable()
		for i := 0; i < n; i++ {
			mfb.Insert(data[i], uint64(i))
		}
		count, size, nnodes := mfb.check(nil)
		assert.T(t).This(count).Is(n)
		full := float32(size) / float32(nnodes) / float32(MaxNodeSize)
		// print("count", count, "nnodes", nnodes, "size", size, "full", full)
		if full < .65 {
			t.Error("expected > .65 got", full)
		}
		mfb.checkData(t, data[:])
	}
	randKey := str.UniqueRandomOf(3, 6, "abcde")
	for i := 0; i < n; i++ {
		data[i] = randKey()
	}
	test()
	sort.Strings(data[:])
	test()
	str.List(data[:]).Reverse()
	test()
}

func (fb *fbtree) checkData(t *testing.T, data []string) {
	t.Helper()
	count, _, _ := fb.check(nil)
	n := 0
	for i, k := range data {
		if data[i] == "" {
			continue
		}
		o := fb.Search(k)
		if o != uint64(i) {
			t.Log("checkData", k, "expect", i, "actual", o)
			t.FailNow()
		}
		n++
	}
	assert.T(t).This(count).Is(n)
}

func TestSampleData(t *testing.T) {
	var nShuffle = 12
	if testing.Short() {
		nShuffle = 4
	}
	test := func(file string) {
		data := fileData(file)
		// fmt.Println(len(data))
		for si := 0; si < nShuffle; si++ {
			rand.Shuffle(len(data),
				func(i, j int) { data[i], data[j] = data[j], data[i] })
			GetLeafKey = func(_ *stor.Stor, _ *ixspec.T, i uint64) string {
				return data[i]
			}
			defer func(mns int) { MaxNodeSize = mns }(MaxNodeSize)
			MaxNodeSize = 256
			fb := CreateFbtree(nil, nil)
			mfb := fb.makeMutable()
			for i, d := range data {
				mfb.Insert(d, uint64(i))
			}
			mfb.checkData(t, data)
		}
	}
	test("../../../bizpartnername.txt")
	test("../../../bizpartnerabbrev.txt")
}

func fileData(filename string) []string {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("can't open", filename)
	}
	defer file.Close()
	data := []string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		data = append(data, scanner.Text())
	}
	return data
}

func TestFbdelete(t *testing.T) {
	var n = 1000
	if testing.Short() {
		n = 100
	}
	data := make([]string, n)
	GetLeafKey = func(_ *stor.Stor, _ *ixspec.T, i uint64) string { return data[i] }
	defer func(mns int) { MaxNodeSize = mns }(MaxNodeSize)
	MaxNodeSize = 44
	fb := CreateFbtree(nil, nil)
	mfb := fb.makeMutable()
	randKey := str.UniqueRandomOf(3, 6, "abcdef")
	for i := 0; i < n; i++ {
		key := randKey()
		data[i] = key
		mfb.Insert(key, uint64(i))
	}
	mfb.checkData(t, data)
	// mfb.print()

	for i := 0; i < len(data); i++ {
		off := rand.Intn(len(data))
		for data[off] == "" {
			off = (off + 1) % len(data)
		}
		// print("================================= delete", data[off])
		mfb.Delete(data[off], uint64(off))
		// mfb.print()
		data[off] = ""
		if i%11 == 0 {
			mfb.checkData(t, data)
		}
	}
}

func TestFreeze(t *testing.T) {
	assert := assert.T(t).This
	GetLeafKey = func(_ *stor.Stor, _ *ixspec.T, i uint64) string {
		return strconv.Itoa(int(i))
	}
	store := stor.HeapStor(8192)
	store.Alloc(1) // avoid offset 0
	fb := CreateFbtree(store, nil)
	assert(fb.redirs.Len()).Is(1)
	fb = fb.Update(func(mfb *fbtree) {
		mfb.Insert("1", 1)
	})
	assert(fb.redirs.Len()).Is(1)
	assert(fb.list()).Is("1")
	fb = fb.Update(func(mfb *fbtree) {
		mfb.Insert("2", 2)
	})
	assert(fb.redirs.Len()).Is(1)
	assert(fb.list()).Is("1 2")

	fb = fb.Save(false)
	fb = OpenFbtree(store, fb.root, fb.treeLevels, fb.redirsOff)
	assert(fb.redirs.Len()).Is(0)
	assert(fb.list()).Is("1 2")
}

func (fb *fbtree) list() string {
	s := ""
	iter := fb.Iter(true)
	for _, o, ok := iter(); ok; _, o, ok = iter() {
		s += strconv.Itoa(int(o)) + " "
	}
	return strings.TrimSpace(s)
}

func TestSave(t *testing.T) {
	var nSaves = 1000
	if testing.Short() {
		nSaves = 100
	}
	const updatesPerSave = 9
	const insertsPerUpdate = 7
	data := make([]string, 0, nSaves*updatesPerSave*insertsPerUpdate)
	GetLeafKey = func(_ *stor.Stor, _ *ixspec.T, i uint64) string { return data[i] }
	defer func(mns int) { MaxNodeSize = mns }(MaxNodeSize)
	MaxNodeSize = 64
	st := stor.HeapStor(8192)
	st.Alloc(1) // avoid offset 0
	fb := CreateFbtree(st, nil)
	randKey := str.UniqueRandomOf(5, 9, "abcdefghi")
	for i := 0; i < nSaves; i++ {
		for j := 0; j < updatesPerSave; j++ {
			fb = fb.Update(func(mfb *fbtree) {
				for k := 0; k < insertsPerUpdate; k++ {
					key := randKey()
					mfb.Insert(key, uint64(len(data)))
					data = append(data, key)
				}
			})
		}
		fb = fb.Save(false) // SAVE
		if i%10 == 9 {
			fb.ckpaths()
			fb.checkData(t, data)
		}
	}
}

func (tbl *RedirHamt) print() {
	fmt.Print("redirs")
	tbl.ForEach(func(r *redir) {
		fmt.Print(" ", OffStr(r.offset)+"->"+OffStr(r.newOffset))
	})
	fmt.Println()
}

func (p *PathHamt) print() {
	fmt.Print("paths")
	p.ForEach(func(o uint64) {
		fmt.Print(" ", OffStr(o))
	})
	fmt.Println()
}

func TestSave2(*testing.T) {
	if testing.Short() {
		return
	}
	const nKeys = 100_000
	keyfn := func(i uint64) string {
		return runtime.Pack(runtime.IntVal(int(i) ^ 0x5555).(runtime.Packable))
	}
	GetLeafKey = func(_ *stor.Stor, _ *ixspec.T, i uint64) string {
		return keyfn(i)
	}
	defer func(mns int) { MaxNodeSize = mns }(MaxNodeSize)
	MaxNodeSize = 128
	st := stor.HeapStor(64 * 1024)
	st.Alloc(1) // avoid offset 0
	fb := CreateFbtree(st, nil).makeMutable()
	for i := uint64(0); i < nKeys; i++ {
		key := keyfn(i)
		fb.Insert(key, uint64(i))
		if i%100 == 99 {
			fb.ckpaths()
			fb = fb.Save(false).makeMutable()
			fb.ckpaths()
		}
	}
	n, _, _ := fb.check(nil)
	assert.This(n).Is(nKeys)
	fb = fb.Save(true)
	n, _, _ = fb.check(nil)
	assert.This(n).Is(nKeys)
}

func TestSplitDup(*testing.T) {
	GetLeafKey = func(_ *stor.Stor, _ *ixspec.T, i uint64) string {
		return strconv.Itoa(int(i))
	}
	defer func(mns int) { MaxNodeSize = mns }(MaxNodeSize)
	MaxNodeSize = 64
	data := []int{}
	for i := 3; i < 8; i++ {
		data = append(data, i)
	}
	for i := 53; i < 58; i++ {
		data = append(data, i)
	}
	for i := 553; i < 558; i++ {
		data = append(data, i)
	}
	for i := 5553; i < 5558; i++ {
		data = append(data, i)
	}
	for i := 55553; i < 55558; i++ {
		data = append(data, i)
	}
	n := 10000
	if testing.Short() {
		n = 1000
	}
	for i := 0; i < n; i++ {
		rand.Shuffle(len(data),
			func(i, j int) { data[i], data[j] = data[j], data[i] })
		fb := CreateFbtree(nil, nil)
		fb = fb.Update(func(mfb *fbtree) {
			for _, n := range data {
				key := strconv.Itoa(n)
				mfb.Insert(key, uint64(n))
			}
		})
	}
}

func TestFlatten(t *testing.T) {
	assert := assert.T(t)
	GetLeafKey = func(_ *stor.Stor, _ *ixspec.T, i uint64) string {
		return strconv.Itoa(int(i))
	}
	defer func(mns int) { MaxNodeSize = mns }(MaxNodeSize)
	const from, to = 10000, 10800
	inserted := map[int]bool{}
	var fb *fbtree

	build := func() {
		_ = T && trace("==============================")
		MaxNodeSize = 96
		inserted = map[int]bool{}
		store := stor.HeapStor(8192)
		bldr := NewFbtreeBuilder(store)
		for i := from; i < to; i += 2 {
			key := strconv.Itoa(i)
			bldr.Add(key, uint64(i))
		}
		fb = bldr.Finish().base()
		assert.That(fb.treeLevels == 2)
		fb.redirs.tbl.ForEach(func(*redir) { panic("redir!") })
		fb.redirs.paths.ForEach(func(uint64) { panic("path!") })
	}
	check := func() {
		t.Helper()
		fb.check(nil)
		iter := fb.Iter(true)
		for i := from; i < to; i++ {
			if i%2 == 1 && !inserted[i] {
				continue
			}
			key := strconv.Itoa(i)
			k, o, ok := iter()
			assert.True(ok)
			assert.True(strings.HasPrefix(key, k))
			assert.This(o).Is(i)
			if o != uint64(i) {
				t.FailNow()
			}
		}
		_, _, ok := iter()
		assert.False(ok)
	}
	insert := func(i int) {
		fb = fb.Update(func(mfb *fbtree) {
			mfb.Insert(strconv.Itoa(i), uint64(i))
			inserted[i] = true
		})
		check()
	}
	maybeSave := func(save bool) {
		check()
		if save {
			fb = fb.Save(false)
			check()
			_ = T && trace("---------------------------")
		}
	}
	flatten := func() {
		fb = fb.Save(true)
		check()
	}

	for _, save := range []bool{false, true} {
		for _, mns := range []int{999, 90} {
			build()
			MaxNodeSize = mns // prevent or force splitting
			insert(10051)
			maybeSave(save)
			flatten()
		}
	}
	for _, save := range []bool{false, true} {
		build()
		MaxNodeSize = 999 // no split
		insert(10051)
		MaxNodeSize = 90 // split all the way
		insert(10551)
		maybeSave(save)
		flatten()
	}
}

//-------------------------------------------------------------------

// ckpaths checks that all the redirects can be reached by following the paths
// and that all paths are in the current tree
func (fb *fbtree) ckpaths() {
	// rset is the set of redirects
	var rset = make(map[uint64]bool)
	fb.redirs.tbl.ForEach(func(r *redir) {
		rset[r.offset] = true
	})
	// pset is the set of paths
	var pset = make(map[uint64]bool)
	fb.redirs.paths.ForEach(func(o uint64) {
		pset[o] = true
	})

	defer func() {
		if e := recover(); e != nil {
			fb.print()
			fmt.Println("root", OffStr(fb.root), "rset", rset)
			fmt.Println("pset", pset)
			fb.printPaths("paths")
			panic(e)
		}
	}()
	delete(rset, fb.root)
	fb.ckpaths1(0, fb.root, true, rset, pset)
	if len(rset) != 0 {
		panic("redirect(s) not found")
	}
	if len(pset) != 0 {
		panic("paths not found")
	}
}

func (fb *fbtree) ckpaths1(depth int, nodeOff uint64, onPath bool,
	rset map[uint64]bool, pset map[uint64]bool) {
	// _, pathNode := fb.redirs.paths.Get(nodeOff)
	pathNode := fb.pathNode(nodeOff)
	if pathNode && !onPath /*&& depth != 1*/ {
		panic("disconnected node in paths")
	}

	delete(pset, nodeOff)
	if depth >= fb.treeLevels {
		// leaf node
		if pathNode && depth != 0 {
			panic("leaf found in paths")
		}
	} else {
		// tree node
		node := fb.getNode(nodeOff)
		for it := node.iter(); it.next(); {
			off := it.offset
			if onPath {
				delete(rset, off)
			}
			_, ok := fb.redirs.tbl.Get(off)
			if ok && !pathNode /*&& depth != 0*/ {
				panic("redirect not on path")
			}
			fb.ckpaths1(depth+1, off, pathNode, rset, pset) // RECURSE
		}
	}
}

func BenchmarkFbtreeInsert(b *testing.B) {
	store := stor.HeapStor(8192)
	bldr := NewFbtreeBuilder(store)
	for i := 100000; i <= 110000; i++ {
		key := strconv.Itoa(i)
		bldr.Add(key, uint64(i))
	}
	fb := bldr.Finish().base()
	GetLeafKey = func(_ *stor.Stor, _ *ixspec.T, i uint64) string {
		return strconv.Itoa(int(i))
	}

	keys := []string{
		"10000", // before first
		"101000a",
		"102000a",
		"103000a",
		"104000a",
		"105000a",
		"106000a",
		"107000a",
		"108000a",
		"109000a",
		"111111", // after last
	}
	for i := 0; i < b.N; i++ {
		// we discard the modified fbtree each time
		fb.Update(func(fb *fbtree) {
			for _, key := range keys {
				fb.Insert(key, 0)
			}
		})
	}
}
