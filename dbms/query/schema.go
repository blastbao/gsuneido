// Copyright Suneido Software Corp. All rights reserved.
// Governed by the MIT license found in the LICENSE file.

package query

import (
	"sort"

	"github.com/apmckinlay/gsuneido/db19/meta"
	"github.com/apmckinlay/gsuneido/db19/meta/schema"
	. "github.com/apmckinlay/gsuneido/runtime"
	"github.com/apmckinlay/gsuneido/util/str"
)

// schema implements pseudo schema tables

type schemaTable struct {
	cache
	tran QueryTran
	state
}

type state int

const (
	rewound state = iota
	within
	eof
)

func (st *schemaTable) Init() {
}

func (st *schemaTable) SetTran(tran QueryTran) {
	st.tran = tran
}

func (st *schemaTable) Indexes() [][]string {
	return nil
}

func (st *schemaTable) Fixed() []Fixed {
	return nil
}

func (st *schemaTable) Updateable() bool {
	return false
}

func (st *schemaTable) SingleTable() bool {
	return false
}

func (st *schemaTable) Output(Record) {
	panic("shouldn't reach here")
}

func (st *schemaTable) optimize(_ Mode, index []string) (Cost, interface{}) {
	if index == nil {
		return 1000, nil
	}
	return impossible, nil
}

func (st *schemaTable) setApproach([]string, interface{}, QueryTran) {
}

func (st *schemaTable) lookupCost() Cost {
	return impossible
}

func (st *schemaTable) Lookup(string) Row {
	panic("shouldn't reach here")
}

func (st *schemaTable) Select(cols, orgs, ends []string) {
	panic("shouldn't reach here")
}

//-------------------------------------------------------------------

type Tables struct {
	schemaTable
	info []*meta.Info
	i    int
}

func (*Tables) String() string {
	return "tables"
}

func (ts *Tables) Transform() Query {
	return ts
}

func (*Tables) Keys() [][]string {
	return [][]string{{"table"}}
}

var tablesFields = [][]string{{"table", "nrows", "totalsize"}}

func (*Tables) Columns() []string {
	return tablesFields[0]
}

func (*Tables) Header() *Header {
	return NewHeader(tablesFields, tablesFields[0])
}

func (ts *Tables) nrows() int {
	ts.ensure()
	return len(ts.info)
}

func (*Tables) rowSize() int {
	return 32 // ???
}

func (ts *Tables) Rewind() {
	ts.i = -1
	ts.state = rewound
}

func (ts *Tables) Get(dir Dir) Row {
	ts.ensure()
	if ts.state == eof {
		return nil
	}
	if dir == Next {
		if ts.state == rewound {
			ts.i = -1
		}
		ts.i++
	} else { // Prev
		if ts.state == rewound {
			ts.i = len(ts.info)
		}
		ts.i--
	}
	if ts.i < 0 || len(ts.info) <= ts.i {
		ts.state = eof
		return nil
	}
	ts.state = within
	info := ts.info[ts.i]
	var rb RecordBuilder
	rb.Add(SuStr(info.Table))
	rb.Add(IntVal(info.Nrows).(Packable))
	rb.Add(Int64Val(int64(info.Size)).(Packable))
	rec := rb.Build()
	return Row{DbRec{Record: rec}}
}

func (ts *Tables) ensure() {
	if ts.info != nil {
		return
	}
	ts.info = ts.tran.GetAllInfo()
	nidxs := 0
	for _, info := range ts.info {
		nidxs += len(info.Indexes)
	}
	ts.info = append(ts.info,
		&meta.Info{Table: "tables", Nrows: len(ts.info) + 3},
		&meta.Info{Table: "columns"},
		&meta.Info{Table: "indexes", Nrows: nidxs},
	)
	sort.Slice(ts.info,
		func(i, j int) bool { return ts.info[i].Table < ts.info[j].Table })
}

//-------------------------------------------------------------------

type Columns struct {
	schemaTable
	state
	schema []*meta.Schema
	si     int
	ci     int
}

func (*Columns) String() string {
	return "columns"
}

func (cs *Columns) Transform() Query {
	return cs
}

func (*Columns) Keys() [][]string {
	return columnsFields
}

var columnsFields = [][]string{{"table", "column"}}

func (cs *Columns) Columns() []string {
	return columnsFields[0]
}

func (*Columns) Header() *Header {
	return NewHeader(columnsFields, columnsFields[0])
}

func (cs *Columns) nrows() int {
	cs.ensure()
	return len(cs.schema)
}

func (*Columns) rowSize() int {
	return 32 // ???
}

func (cs *Columns) Rewind() {
	cs.si = -1
	cs.state = rewound
}

func (cs *Columns) Get(dir Dir) Row {
	cs.ensure()
	if cs.state == eof {
		return nil
	}
	if dir == Next {
		if cs.state == rewound {
			cs.si, cs.ci = 0, 0
		} else {
			cs.ci++
			if cs.ci >= len(cs.schema[cs.si].Columns) {
				cs.si++
				if cs.si >= len(cs.schema) {
					cs.state = eof
					return nil
				}
				cs.ci = 0
			}
		}
	} else { // Prev
		if cs.state == rewound {
			cs.si = len(cs.schema) - 1
			cs.ci = len(cs.schema[cs.si].Columns) - 1
		} else {
			cs.ci--
			if cs.ci < 0 {
				cs.si--
				if cs.si < 0 {
					cs.state = eof
					return nil
				}
				cs.ci = len(cs.schema[cs.si].Columns) - 1
			}
		}
	}
	cs.state = within
	schema := cs.schema[cs.si]
	var rb RecordBuilder
	rb.Add(SuStr(schema.Table))
	rb.Add(SuStr(schema.Columns[cs.ci]))
	rec := rb.Build()
	return Row{DbRec{Record: rec}}
}

func (cs *Columns) ensure() {
	if cs.schema != nil {
		return
	}
	cs.schema = cs.tran.GetAllSchema()
	cs.schema = append(cs.schema,
		&meta.Schema{Schema: schema.Schema{Table: "tables", Columns: tablesFields[0]}},
		&meta.Schema{Schema: schema.Schema{Table: "columns", Columns: columnsFields[0]}},
		&meta.Schema{Schema: schema.Schema{Table: "indexes", Columns: indexesFields[0]}},
	)
	sort.Slice(cs.schema,
		func(i, j int) bool { return cs.schema[i].Table < cs.schema[j].Table })
}

//-------------------------------------------------------------------

type Indexes struct {
	schemaTable
	state
	schema []*meta.Schema
	si     int
	ci     int
}

func (*Indexes) String() string {
	return "indexes"
}

func (is *Indexes) Transform() Query {
	return is
}

func (*Indexes) Keys() [][]string {
	return [][]string{{"table", "columns"}}
}

var indexesFields = [][]string{{"table", "columns", "key"}}

func (is *Indexes) Columns() []string {
	return indexesFields[0]
}

func (*Indexes) Header() *Header {
	return NewHeader(indexesFields, indexesFields[0])
}

func (is *Indexes) nrows() int {
	is.ensure()
	return len(is.schema)
}

func (*Indexes) rowSize() int {
	return 32 // ???
}

func (is *Indexes) Rewind() {
	is.si = -1
	is.state = rewound
}

func (is *Indexes) Get(dir Dir) Row {
	is.ensure()
	if is.state == eof {
		return nil
	}
	if dir == Next {
		if is.state == rewound {
			is.si, is.ci = 0, 0
		} else {
			is.ci++
			if is.ci >= len(is.schema[is.si].Indexes) {
				is.si++
				if is.si >= len(is.schema) {
					is.state = eof
					return nil
				}
				is.ci = 0
			}
		}
	} else { // Prev
		if is.state == rewound {
			is.si = len(is.schema) - 1
			is.ci = len(is.schema[is.si].Indexes) - 1
		} else {
			is.ci--
			if is.ci < 0 {
				is.si--
				if is.si < 0 {
					is.state = eof
					return nil
				}
				is.ci = len(is.schema[is.si].Indexes) - 1
			}
		}
	}
	is.state = within
	schema := is.schema[is.si]
	var rb RecordBuilder
	rb.Add(SuStr(schema.Table))
	idx := schema.Indexes[is.ci]
	rb.Add(SuStr(str.Join(",", idx.Columns)))
	switch idx.Mode {
	case 'k':
		rb.Add(True.(Packable))
	case 'i':
		rb.Add(False.(Packable))
	case 'u':
		rb.Add(SuStr("u"))
	default:
		panic("shouldn't reach here")
	}
	rec := rb.Build()
	return Row{DbRec{Record: rec}}
}

func (is *Indexes) ensure() {
	if is.schema != nil {
		return
	}
	is.schema = is.tran.GetAllSchema()
	sort.Slice(is.schema,
		func(i, j int) bool { return is.schema[i].Table < is.schema[j].Table })
}