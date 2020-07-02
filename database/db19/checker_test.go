// Copyright Suneido Software Corp. All rights reserved.
// Governed by the MIT license found in the LICENSE file.

package db19

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/apmckinlay/gsuneido/util/verify"
)

func TestCheckerStartStop(*testing.T) {
	ck := NewCheck()
	const ntrans = 20
	var trans [ntrans]int
	const ntimes = 5000
	for i := 0; i < ntimes; i++ {
		j := rand.Intn(ntrans)
		if trans[j] == 0 {
			trans[j] = ck.StartTran()
		} else {
			if rand.Intn(2) == 1 {
				ck.Commit(trans[j])
			} else {
				ck.Abort(trans[j])
			}
			trans[j] = 0
		}
	}
	for _, tn := range trans {
		if tn != 0 {
			ck.Commit(tn)
		}
	}
	verify.That(len(ck.trans) == 0)
}

func TestCheckerActions(t *testing.T) {
	// writes
	script(t, "1w1 2w2 1c 2c")
	script(t, "1w4 1w5 2w6 2w7 1c 2c")
	script(t, "1w1 2w2 1c 2a")
	script(t, "1w1 2w2 1a 2c")
	script(t, "1w1 2w2 1a 2a")
	// conflict
	script(t, "1w1 1w2 2w3 2w1 1a 2a")
	script(t, "1w1 2w1 1a 2c")
	script(t, "1w1 2w1 1c 2C")
	script(t, "1w1 2w1 1c 2A")
	script(t, "1w4 1w5 2w3 2w5 1c 2A")
	// conflict with ended
	script(t, "1w1 1c 2W1 2C")
	script(t, "2w1 2c 1W1 1C")
}

func script(t *testing.T, s string) {
	ok := func(result bool) {
		if result != true {
			t.Log("incorrect at:", s)
			t.FailNow()
		}
	}
	fail := func(result bool) {
		if result != false {
			t.Log("incorrect at:", s)
			t.FailNow()
		}
	}
	ck := NewCheck()
	ts := []int{ck.StartTran(), ck.StartTran()}
	for len(s) > 0 {
		t := ts[s[0]-'1']
		switch s[1] {
		case 'w':
			ok(ck.Write(t, "mytable", []string{"", s[2:3]}))
			s = s[1:]
		case 'W':
			fail(ck.Write(t, "mytable", []string{"", s[2:3]}))
			s = s[1:]
		case 'c':
			ok(ck.Commit(t))
		case 'C':
			fail(ck.Commit(t))
		case 'a':
			ok(ck.Abort(t))
		case 'A':
			fail(ck.Abort(t))
		}
		s = s[2:]
		for len(s) > 0 && s[0] == ' ' {
			s = s[1:]
		}
	}
}

func (t *cktran) String() string {
	b := new(bytes.Buffer)
	fmt.Fprint(b, "T", t.start)
	if t.isEnded() {
		fmt.Fprint(b, "->", t.end)
	}
	fmt.Fprintln(b)
	for name, tbl := range t.tables {
		fmt.Fprintln(b, "    ", name)
		for i, set := range tbl.writes {
			if set != nil {
				fmt.Fprintln(b, "        index", i, ":", set.String())
			}
		}
	}
	return strings.TrimSpace(b.String())
}
