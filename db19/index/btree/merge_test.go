// Copyright Suneido Software Corp. All rights reserved.
// Governed by the MIT license found in the LICENSE file.

package btree

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"

	"github.com/apmckinlay/gsuneido/db19/index/ixbuf"
	"github.com/apmckinlay/gsuneido/db19/index/ixkey"
	"github.com/apmckinlay/gsuneido/db19/index/testdata"
	"github.com/apmckinlay/gsuneido/db19/stor"
	"github.com/apmckinlay/gsuneido/util/assert"
)

func TestMergeDeleteAll(*testing.T) {
	GetLeafKey = func(_ *stor.Stor, _ *ixkey.Spec, i uint64) string {
		return strconv.Itoa(int(i))
	}
	defer func(mns int) { MaxNodeSize = mns }(MaxNodeSize)
	MaxNodeSize = 64

	org, end := 100, 999
	bldr := Builder(stor.HeapStor(8192))
	for i := org; i < end; i++ {
		key := strconv.Itoa(i)
		bldr.Add(key, uint64(i))
	}
	bt := bldr.Finish()
	// bt.Print()

	ib := &ixbuf.T{}
	for i := org; i < end; i++ {
		key := strconv.Itoa(i)
		ib.Insert(key, ixbuf.Delete|uint64(i))
	}

	out := bt.MergeAndSave(ib.Iter())
	// out.Print()

	iter := out.Iterator()
	iter.Next()
	assert.That(iter.Eof())
}

func TestMergeAndSave(*testing.T) {
	nMerges := 2000
	opsPerMerge := 1000
	if testing.Short() {
		nMerges = 200
		opsPerMerge = 200
	}
	d := testdata.New()
	GetLeafKey = d.GetLeafKey
	defer func(mns int) { MaxNodeSize = mns }(MaxNodeSize)
	MaxNodeSize = 64
	bt := CreateBtree(stor.HeapStor(8192), nil)

	for i := 0; i < nMerges; i++ {
		_ = t && trace("---")
		x := &ixbuf.T{}
		for j := 0; j < opsPerMerge; j++ {
			k := rand.Intn(4)
			switch {
			case k == 0 || k == 1 || d.Len() == 0:
				x.Insert(d.Gen())
			case k == 2:
				_, key, _ := d.Rand()
				off := d.NextOff()
				x.Update(key, off)
				d.Update(key, off)
			case k == 3:
				i, key, off := d.Rand()
				x.Delete(key, off)
				d.Delete(i)
			}
		}
		bt = bt.MergeAndSave(x.Iter())
	}
	bt.Check(nil)
	d.Check(bt)
	d.CheckIter(bt.Iterator())
}

func TestBtreePrefixExists(*testing.T) {
	defer func(mns int) { MaxNodeSize = mns }(MaxNodeSize)
	MaxNodeSize = 200
	key := func(i int) string {
		s := "1"
		if i >= 16 {
			s = "2"
		}
		s += "\x00\x00" + fmt.Sprintf("%02d", i)
		return s
	}
	GetLeafKey = func(_ *stor.Stor, _ *ixkey.Spec, i uint64) string {
		return key(int(i))
	}
	b := Builder(stor.HeapStor(8192))
	for i := 0; i < 22; i++ {
		b.Add(key(i), uint64(i))
	}
	bt := b.Finish()

	ib := &ixbuf.T{}
	ib.Delete(key(16), 16)
	bt = bt.MergeAndSave(ib.Iter())
	// bt.Print()
	// should be like:
	// . "1|15"
	// 7: 0 2|17
	// . "2|17"

	assert.That(bt.PrefixExists("2"))
}

//-------------------------------------------------------------------

func (st *state) print() {
	fmt.Println("state:", st.bt.treeLevels)
	for _, m := range st.path {
		fmt.Println("   ", &m)
		fmt.Println("       ", m.node.knowns())
	}
}

func (m *merge) String() string {
	limit := m.limit
	if limit == "" {
		limit = `""`
	}
	mod := ""
	if m.modified {
		mod = " modified"
	}
	return fmt.Sprint("off ", m.off, " pos ", m.pos, " limit ", limit, mod)
}
