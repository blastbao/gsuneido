// Copyright Suneido Software Corp. All rights reserved.
// Governed by the MIT license found in the LICENSE file.

package runtime

import (
	"fmt"
	"strings"

	"github.com/apmckinlay/gsuneido/runtime/trace"
	"github.com/apmckinlay/gsuneido/runtime/types"
	"github.com/apmckinlay/gsuneido/util/assert"
	"github.com/apmckinlay/gsuneido/util/pack"
	"github.com/apmckinlay/gsuneido/util/str"
	"github.com/apmckinlay/gsuneido/util/strs"
)

// SuRecord is an SuObject with observers and rules and a default value of "".
// Uses the lock from SuObject.
type SuRecord struct {
	ob SuObject
	CantConvert
	// observers is from record.Observer(fn)
	observers ValueList
	// invalidated accumulates keys needing observers called
	invalidated str.Queue
	// invalid is the fields that need to be recalculated
	invalid map[string]bool
	// dependents are the fields that depend on a field
	dependents map[string][]string
	// activeObservers is used to prevent infinite recursion
	activeObservers ActiveObserverList
	// attachedRules is from record.AttachRule(key,fn)
	attachedRules map[string]Value

	// row is used when it is from the database
	row Row
	// header is the Header for row
	hdr *Header
	// tran is the database transaction used to read the record
	tran *SuTran
	// recoff is the record offset in the database
	recoff uint64
	// table is the table the record came from if it's updateable, else ""
	table string
	// status
	status Status
	// userow is true when we want to use data in row as well as ob
	userow bool
}

type Status byte

const (
	NEW Status = iota
	OLD
	DELETED
)

//go:generate genny -in ../genny/list/list.go -out alist.go -pkg runtime gen "V=activeObserver"
//go:generate genny -in ../genny/list/list.go -out vlist.go -pkg runtime gen "V=Value"

func NewSuRecord() *SuRecord {
	return &SuRecord{ob: SuObject{defval: EmptyStr}}
}

// SuRecordFromObject creates a record from an arguments object
// WARNING: it does not copy the data, the original object should be discarded
func SuRecordFromObject(ob *SuObject) *SuRecord {
	return &SuRecord{
		ob: SuObject{list: ob.list, named: ob.named, defval: EmptyStr}}
}

func SuRecordFromRow(row Row, hdr *Header, table string, tran *SuTran) *SuRecord {
	rec := SuRecord{row: row, hdr: hdr, tran: tran,
		ob: SuObject{defval: EmptyStr}, userow: true, status: OLD}
	if table != "" {
		rec.table = table
		rec.recoff = row[0].Off
	}
	return &rec
}

func (r *SuRecord) ensureDeps() {
	if r.dependents == nil && r.row != nil {
		r.dependents = deps(r.row, r.hdr)
	}
}

func deps(row Row, hdr *Header) map[string][]string {
	dependents := map[string][]string{}
	for _, flds := range hdr.Fields {
		for _, f := range flds {
			if strings.HasSuffix(f, "_deps") {
				val := Unpack(row.GetRaw(hdr, f))
				deps := str.Split(ToStr(val), ",")
				f = f[:len(f)-5]
				for _, d := range deps {
					if !strs.Contains(dependents[d], f) {
						dependents[d] = append(dependents[d], f)
					}
				}
			}
		}
	}
	return dependents
}

func (r *SuRecord) Copy() Container {
	if r.ob.Lock() {
		defer r.ob.Unlock()
	}
	return r.slice(0)
}

// slice returns a copy of a record, omitting the first n list values
func (r *SuRecord) slice(n int) *SuRecord {
	// keep row and hdr even if unpacked, to help ToRecord
	return &SuRecord{
		ob:         *r.ob.slice(n),
		row:        r.row,
		hdr:        r.hdr,
		userow:     r.userow,
		status:     r.status,
		dependents: r.copyDeps(),
		invalid:    r.copyInvalid()}
}

func (r *SuRecord) copyDeps() map[string][]string {
	copy := make(map[string][]string, len(r.dependents))
	for k, v := range r.dependents {
		copy[k] = append(v[:0:0], v...) // copy slice
	}
	return copy
}

func (r *SuRecord) copyInvalid() map[string]bool {
	copy := make(map[string]bool, len(r.invalid))
	for k, v := range r.invalid {
		copy[k] = v
	}
	return copy
}

func (*SuRecord) Type() types.Type {
	return types.Record
}

func (r *SuRecord) String() string {
	return r.Display(nil)
}

func (r *SuRecord) Display(t *Thread) string {
	buf := limitBuf{}
	r.rstring(t, &buf, nil)
	return buf.String()
}

func (r *SuRecord) rstring(t *Thread, buf *limitBuf, inProgress vstack) {
	r.ToObject().rstring2(t, buf, "[", "]", inProgress)
}

var _ recursable = (*SuRecord)(nil)

func (r *SuRecord) Show() string {
	return r.ToObject().show("[", "]", nil)
}

func (*SuRecord) Call(*Thread, Value, *ArgSpec) Value {
	panic("can't call Record")
}

func (r *SuRecord) Compare(other Value) int {
	return r.ToObject().Compare(other)
}

func (r *SuRecord) Equal(other interface{}) bool {
	return r.ToObject().Equal(other) //FIXME not symmetrical with SuObject.Equal
}

func (r *SuRecord) Hash() uint32 {
	return r.ToObject().Hash()
}

func (r *SuRecord) Hash2() uint32 {
	return r.ToObject().Hash2()
}

func (r *SuRecord) RangeTo(from int, to int) Value {
	return r.ob.RangeTo(from, to)
}

func (r *SuRecord) RangeLen(from int, n int) Value {
	return r.ob.RangeLen(from, n)
}

func (r *SuRecord) ToContainer() (Container, bool) {
	return r, true
}

func (r *SuRecord) SetConcurrent() {
	r.ob.SetConcurrent()
}
func (r *SuRecord) Lock() bool {
	return r.ob.Lock()
}
func (r *SuRecord) Unlock() bool {
	return r.ob.Unlock()
}
func (r *SuRecord) IsConcurrent() Value {
	return r.ob.IsConcurrent()
}

// Container --------------------------------------------------------

var _ Container = (*SuRecord)(nil) // includes Value and Lockable

func (r *SuRecord) ToObject() *SuObject {
	if r.ob.Lock() {
		defer r.ob.Unlock()
	}
	return r.toObject()
}
func (r *SuRecord) toObject() *SuObject {
	if r.userow {
		for ri, rf := range r.hdr.Fields {
			for fi, f := range rf {
				if f != "-" && !strings.HasSuffix(f, "_deps") {
					key := SuStr(f)
					if !r.ob.hasKey(key) {
						if val := r.row[ri].GetRaw(fi); val != "" {
							r.ob.set(key, Unpack(val))
						}
					}
				}
			}
		}
		r.userow = false
		// keep row for ToRecord
	}
	return &r.ob
}

func (r *SuRecord) Add(val Value) {
	r.ob.Add(val)
}

func (r *SuRecord) Insert(at int, val Value) {
	r.ob.Insert(at, val)
}

func (r *SuRecord) HasKey(key Value) bool {
	if r.ob.Lock() {
		defer r.ob.Unlock()
	}
	if r.ob.hasKey(key) {
		return true
	}
	if r.userow {
		if k, ok := key.ToStr(); ok {
			return r.row.GetRaw(r.hdr, k) != ""
		}
	}
	return false
}

func (r *SuRecord) Set(key, val Value) {
	r.Put(nil, key, val)
}

func (r *SuRecord) Clear() {
	if r.ob.Lock() {
		defer r.ob.Unlock()
	}
	r.ob.mustBeMutable()
	*r = *NewSuRecord()
}

func (r *SuRecord) DeleteAll() {
	if r.ob.Lock() {
		defer r.ob.Unlock()
	}
	r.ob.deleteAll()
	r.row = nil
	r.userow = false
}

func (r *SuRecord) SetReadOnly() {
	// unpack fully before setting readonly
	// because lazy unpack will no longer be able to save values
	r.ToObject()
	r.ob.SetReadOnly()
}

func (r *SuRecord) IsReadOnly() bool {
	return r.ob.IsReadOnly()
}

func (r *SuRecord) isReadOnly() bool {
	return r.ob.isReadOnly()
}

func (r *SuRecord) IsNew() bool {
	return r.status == NEW
}

func (r *SuRecord) Delete(t *Thread, key Value) bool {
	return r.delete(t, key, r.ob.delete)
}

func (r *SuRecord) Erase(t *Thread, key Value) bool {
	return r.delete(t, key, r.ob.erase)
}

func (r *SuRecord) delete(t *Thread, key Value, fn func(Value) bool) bool {
	if r.ob.Lock() {
		defer r.ob.Unlock()
	}
	r.ensureDeps()
	r.ob.mustBeMutable()
	// have to unpack
	// because we have no way to delete from row
	r.toObject()
	// have to remove row
	// because we assume if field is missing from object we can use row data
	r.row = nil
	if fn(key) {
		if keystr, ok := key.ToStr(); ok {
			r.invalidateDependents(keystr)
			r.callObservers(t, keystr)
		}
		return true
	}
	return false
}

func (r *SuRecord) ListSize() int {
	return r.ob.ListSize()
}

func (r *SuRecord) ListGet(i int) Value {
	return r.ob.ListGet(i)
}

func (r *SuRecord) NamedSize() int {
	if r.ob.Lock() {
		defer r.ob.Unlock()
	}
	if r.userow {
		return r.rowSize()
	}
	return r.ob.named.Size()
}

func (r *SuRecord) rowSize() int {
	n := r.ob.named.Size()
	for ri, rf := range r.hdr.Fields {
		for fi, f := range rf {
			if f != "-" && !strings.HasSuffix(f, "_deps") {
				key := SuStr(f)
				if !r.ob.hasKey(key) {
					if val := r.row[ri].GetRaw(fi); val != "" {
						n++
					}
				}
			}
		}
	}
	return n
}

func (r *SuRecord) ArgsIter() func() (Value, Value) {
	return r.ToObject().ArgsIter()
}

func (r *SuRecord) Iter2(list bool, named bool) func() (Value, Value) {
	return r.ToObject().Iter2(list, named)
}

func (r *SuRecord) Slice(n int) Container {
	if r.ob.Lock() {
		defer r.ob.Unlock()
	}
	return r.slice(n)
}

func (r *SuRecord) Iter() Iter {
	if r.ob.Lock() {
		defer r.ob.Unlock()
	}
	r.toObject()
	return &obIter{ob: &r.ob, iter: r.ob.iter2(true, true),
		result: func(k, v Value) Value { return v }}
}

// ------------------------------------------------------------------

func (r *SuRecord) Put(t *Thread, keyval, val Value) {
	if r.Lock() {
		defer r.Unlock()
	}
	r.put(t, keyval, val)
}
func (r *SuRecord) put(t *Thread, keyval, val Value) {
	r.trace("Put", keyval, "=", val)
	r.ensureDeps()
	if key, ok := keyval.ToStr(); ok {
		delete(r.invalid, key)
		old := r.ob.getIfPresent(keyval)
		if old == nil && r.userow {
			old = r.getFromRow(key)
		}
		r.ob.set(keyval, val)
		if old != nil && r.same(old, val) {
			return
		}
		r.invalidateDependents(key)
		r.callObservers(t, key)
	} else { // key not a string
		r.ob.set(keyval, val)
	}
}

func (*SuRecord) same(x, y Value) bool {
	// only use Equal with simple values to avoid deadlock
	if x.Type() <= types.Date && y.Type() <= types.Date {
		return x.Equal(y) // compare by value
	}
	return x == y // compare by reference
}

func (r *SuRecord) invalidateDependents(key string) {
	r.trace("invalidate dependents of", key)
	for _, d := range r.dependents[key] {
		r.invalidate(d)
	}
}

func (r *SuRecord) GetPut(t *Thread, m, v Value,
	op func(x, y Value) Value, retOrig bool) Value {
	if r.Lock() {
		defer r.Unlock()
	}
	orig := r.get(t, m)
	if orig == nil {
		panic("uninitialized member: " + m.String())
	}
	v = op(orig, v)
	r.put(t, m, v)
	if retOrig {
		return orig
	}
	return v
}

func (r *SuRecord) Invalidate(t *Thread, key string) {
	if r.Lock() {
		defer r.Unlock()
	}
	r.ensureDeps()
	r.invalidate(key)
	r.callObservers(t, key)
}

func (r *SuRecord) invalidate(key string) {
	if r.invalid[key] {
		return
	}
	r.invalidated.Add(key) // for observers
	if r.invalid == nil {
		r.invalid = make(map[string]bool)
	}
	r.trace("invalidate", key)
	r.invalid[key] = true
	r.invalidateDependents(key)
}

func (r *SuRecord) PreSet(key, val Value) {
	r.ob.Set(key, val)
}

func (r *SuRecord) Observer(ofn Value) {
	if r.Lock() {
		defer r.Unlock()
	}
	r.observers.Push(ofn)
}

func (r *SuRecord) RemoveObserver(ofn Value) bool {
	if r.Lock() {
		defer r.Unlock()
	}
	return r.observers.Remove(ofn)
}

func (r *SuRecord) callObservers(t *Thread, key string) {
	r.callObservers2(t, key)
	for !r.invalidated.Empty() {
		if k := r.invalidated.Take(); k != key {
			r.callObservers2(t, k)
		}
	}
}

func (r *SuRecord) callObservers2(t *Thread, key string) {
	for _, x := range r.observers.list {
		ofn := x.(Value)
		if !r.activeObservers.Has(activeObserver{ofn, key}) {
			func(ofn Value, key string) {
				r.activeObservers.Push(activeObserver{ofn, key})
				defer r.activeObservers.Pop()
				func() {
					if r.Unlock() { // can't hold lock while calling observer
						defer r.Lock()
					}
					t.PushCall(ofn, r, argSpecMember, SuStr(key))
				}()
			}(ofn, key)
		}
	}
}

var argSpecMember = &ArgSpec{Nargs: 1,
	Spec: []byte{0}, Names: []Value{SuStr("member")}}

type activeObserver struct {
	obs Value
	key string
}

func (a activeObserver) Equal(other activeObserver) bool {
	return a.key == other.key && a.obs.Equal(other.obs)
}

// ------------------------------------------------------------------

// Get returns the value associated with a key, or defval if not found
func (r *SuRecord) Get(t *Thread, key Value) Value {
	if r.Lock() {
		defer r.Unlock()
	}
	return r.get(t, key)
}
func (r *SuRecord) get(t *Thread, key Value) Value {
	r.trace("Get", key)
	if val := r.getIfPresent(t, key); val != nil {
		return val
	}
	return r.ob.defaultValue(key)
}

// GetIfPresent is the same as Get
// except it returns nil instead of defval for missing members
func (r *SuRecord) GetIfPresent(t *Thread, keyval Value) Value {
	if r.Lock() {
		defer r.Unlock()
	}
	return r.getIfPresent(t, keyval)
}
func (r *SuRecord) getIfPresent(t *Thread, keyval Value) Value {
	result := r.ob.getIfPresent(keyval)
	if key, ok := keyval.ToStr(); ok {
		// only do record stuff when key is a string
		if t != nil {
			if ar := t.rules.top(); ar.rec == r { // identity (not Equal)
				r.addDependent(ar.key, key)
			}
		}
		if result == nil && r.userow {
			if x := r.getFromRow(key); x != nil {
				result = x
			}
		}
		if result == nil || r.invalid[key] {
			if x := r.getSpecial(key); x != nil {
				result = x
			} else if x = r.callRule(t, key); x != nil {
				result = x
			}
		}
	}
	return result
}

func (r *SuRecord) getFromRow(key string) Value {
	if raw := r.row.GetRaw(r.hdr, key); raw != "" {
		val := Unpack(raw)
		if !r.ob.readonly {
			r.ob.set(SuStr(key), val) // cache unpacked value
		}
		return val
	}
	return nil
}

// deps is used by ToRecord to create the dependencies for a field
// but without unpacking if it's in the row
func (r *SuRecord) deps(t *Thread, key string) {
	result := r.ob.getIfPresent(SuStr(key))
	if t != nil {
		if ar := t.rules.top(); ar.rec == r { // identity (not Equal)
			r.addDependent(ar.key, key)
		}
	}
	if result == nil && r.userow {
		if s := r.row.GetRaw(r.hdr, key); s != "" {
			result = True
		}
	}
	if result == nil || r.invalid[key] {
		if x := r.getSpecial(key); x == nil {
			r.callRule(t, key)
		}
	}
}

// getPacked is used by ToRecord to build a Record for the database.
// It is like Get except it returns the value packed,
// using the already packed value from the row when possible.
// It does not add dependencies or handle special fields (e.g. _lower!)
func (r *SuRecord) getPacked(t *Thread, key string) string {
	result := r.ob.getIfPresent(SuStr(key)) // NOTE: ob.getIfPresent
	packed := ""
	if result == nil && r.row != nil { // even if !r.userow
		if s := r.row.GetRaw(r.hdr, key); s != "" {
			packed = s
			result = True
		}
	}
	if result == nil || r.invalid[key] {
		if x := r.callRule(t, key); x != nil {
			result = x
		}
	}
	if result == nil {
		result = r.ob.defval
		if result == nil {
			return ""
		}
	}
	if packed != "" {
		return packed
	}
	return PackValue(result)
}

func (r *SuRecord) addDependent(from, to string) {
	if from == to {
		return
	}
	if r.dependents == nil {
		r.dependents = make(map[string][]string)
	}
	if !strs.Contains(r.dependents[to], from) {
		r.trace("add dependency for", from, "uses", to)
		r.dependents[to] = append(r.dependents[to], from)
	}
}

// getSpecial handles _lower!
func (r *SuRecord) getSpecial(key string) Value {
	if strings.HasSuffix(key, "_lower!") {
		basekey := key[0 : len(key)-7]
		if val := r.getIfPresent(nil, SuStr(basekey)); val != nil {
			if vs, ok := val.ToStr(); ok {
				val = SuStr(str.ToLower(vs))
			}
			return val
		}
	}
	return nil
}

func (r *SuRecord) callRule(t *Thread, key string) Value {
	r.ensureDeps()
	delete(r.invalid, key)
	if rule := r.getRule(t, key); rule != nil && !t.rules.has(r, key) {
		r.trace("call rule", key)
		val := r.catchRule(t, rule, key)
		if val != nil && !r.ob.readonly {
			r.ob.set(SuStr(key), val)
		}
		return val
	}
	return nil
}

func (r *SuRecord) catchRule(t *Thread, rule Value, key string) Value {
	t.rules.push(r, key)
	defer func() {
		t.rules.pop()
		if e := recover(); e != nil {
			WrapPanic(e, "rule for "+key)
		}
	}()
	if r.Unlock() { // can't hold lock while calling observers
		defer r.Lock()
	}
	return t.CallThis(rule, r)
}

func WrapPanic(e interface{}, suffix string) {
	switch e := e.(type) {
	case *SuExcept:
		s := string(e.SuStr) + " (" + suffix + ")"
		panic(&SuExcept{SuStr: SuStr(s), Callstack: e.Callstack})
	case error:
		panic(fmt.Errorf("%w (%s)", e, suffix))
	case Value:
		panic(ToStrOrString(e) + " (" + suffix + ")")
	default:
		panic(fmt.Sprint(e) + " (" + suffix + ")")
	}
}

// activeRules stack
type activeRules struct {
	list []activeRule
}
type activeRule struct {
	rec *SuRecord
	key string
}

func (ar *activeRules) push(r *SuRecord, key string) {
	ar.list = append(ar.list, activeRule{r, key})
}
func (ar *activeRules) top() activeRule {
	if len(ar.list) == 0 {
		return activeRule{}
	}
	return ar.list[len(ar.list)-1]
}
func (ar *activeRules) pop() {
	ar.list = ar.list[:len(ar.list)-1]
}
func (ar *activeRules) has(r *SuRecord, key string) bool {
	for _, x := range ar.list {
		if x.rec == r && x.key == key { // record identity (not Equal)
			return true
		}
	}
	return false
}

func (r *SuRecord) getRule(t *Thread, key string) Value {
	if rule, ok := r.attachedRules[key]; ok {
		assert.That(rule != nil)
		return rule
	}
	//TODO key should be identifier but we have foobar?__protect
	if r.ob.defval != nil && t != nil && key != "" {
		return Global.FindName(t, "Rule_"+key)
	}
	return nil
}

func (r *SuRecord) AttachRule(key, callable Value) {
	if r.Lock() {
		defer r.Unlock()
	}
	if r.attachedRules == nil {
		r.attachedRules = make(map[string]Value)
	}
	r.attachedRules[AsStr(key)] = callable
}

func (r *SuRecord) GetDeps(key string) Value {
	if r.Lock() {
		defer r.Unlock()
	}
	r.ensureDeps()
	var sb strings.Builder
	sep := ""
	for to, froms := range r.dependents {
		for _, from := range froms {
			if from == key {
				sb.WriteString(sep)
				sb.WriteString(to)
				sep = ","
			}
		}
	}
	return SuStr(sb.String())
}

func (r *SuRecord) SetDeps(key, deps string) {
	if r.Lock() {
		defer r.Unlock()
	}
	if deps == "" {
		return
	}
	r.ensureDeps()
outer:
	for _, to := range strings.Split(deps, ",") {
		to = strings.TrimSpace(to)
		for _, from := range r.dependents[to] {
			if from == key {
				continue outer
			}
		}
		r.addDependent(key, to)
	}
}

func (r *SuRecord) Transaction() *SuTran {
	if r.Lock() {
		defer r.Unlock()
	}
	return r.tran
}

// ToRecord converts this SuRecord to a Record to be stored in the database
func (r *SuRecord) ToRecord(t *Thread, hdr *Header) Record {
	if r.Lock() {
		defer r.Unlock()
	}
	r.ensureDeps()
	fields := hdr.Fields[0]

	// ensure dependencies are created
	for _, f := range fields {
		if f != "-" {
			r.deps(t, f)
		}
	}

	// invert stored dependencies
	deps := map[string][]string{}
	for k, v := range r.dependents {
		for _, d := range v {
			d_deps := d + "_deps"
			if strs.Contains(fields, d_deps) {
				deps[d_deps] = append(deps[d_deps], k)
			}
		}
	}

	rb := RecordBuilder{}
	var tsField string
	var ts SuDate
	for _, f := range fields {
		if f == "-" {
			rb.AddRaw("")
		} else if strings.HasSuffix(f, "_TS") { // also done in SuObject ToRecord
			if tsField != "" {
				panic("multiple _TS fields not supported")
			}
			tsField = f
			ts = t.Dbms().Timestamp()
			rb.Add(ts)
		} else if d, ok := deps[f]; ok {
			rb.Add(SuStr(strings.Join(d, ",")))
		} else {
			rb.AddRaw(r.getPacked(t, f))
		}
	}
	if tsField != "" && !r.isReadOnly() {
		r.ob.set(SuStr(tsField), ts) // NOTE: ob.set
	}
	return rb.Trim().Build()
}

// RecordMethods is initialized by the builtin package
var RecordMethods Methods

var gnRecords = Global.Num("Records")

func (*SuRecord) Lookup(t *Thread, method string) Callable {
	if m := Lookup(t, RecordMethods, gnRecords, method); m != nil {
		return m
	}
	return (*SuObject)(nil).Lookup(t, method)
}

// Packable ---------------------------------------------------------

//TODO use row

var _ Packable = (*SuRecord)(nil)

func (r *SuRecord) PackSize(clock *int32) int {
	return r.ToObject().PackSize(clock)
}

func (r *SuRecord) PackSize2(clock int32, stack packStack) int {
	return r.ToObject().PackSize2(clock, stack)
}

func (r *SuRecord) PackSize3() int {
	return r.ToObject().PackSize3()
}

func (r *SuRecord) Pack(clock int32, buf *pack.Encoder) {
	r.ToObject().pack(clock, buf, PackRecord)
}

func UnpackRecord(s string) *SuRecord {
	r := NewSuRecord()
	unpackObject(s, &r.ob)
	return r
}

// database

func (r *SuRecord) DbDelete() {
	if r.Lock() {
		defer r.Unlock()
	}
	r.ckModify("Delete")
	r.tran.Delete(r.table, r.recoff)
	r.status = DELETED
}

func (r *SuRecord) DbUpdate(t *Thread, ob Value) {
	var rec Record
	if ob == False {
		rec = r.ToRecord(t, r.hdr)
	} else {
		rec = ToContainer(ob).ToRecord(t, r.hdr)
	}
	if r.Lock() {
		defer r.Unlock()
	}
	r.ckModify("Update")
	r.recoff = r.tran.Update(r.table, r.recoff, rec) // ??? ok while locked ???
}

func (r *SuRecord) ckModify(op string) {
	if r.tran == nil {
		panic("record." + op + ": no Transaction")
	}
	if r.status != OLD || r.recoff == 0 {
		panic("record." + op + ": not a database record")
	}
}

func (r *SuRecord) trace(args ...interface{}) {
	// %p is the address of the record
	trace.Records.Println(fmt.Sprintf("%p ", r), args...)
}
