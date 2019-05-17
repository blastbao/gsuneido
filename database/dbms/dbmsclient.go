package dbms

import (
	"io"
	"net"
	"strconv"
	"strings"

	"github.com/apmckinlay/gsuneido/database/dbms/commands"
	"github.com/apmckinlay/gsuneido/database/dbms/csio"
	. "github.com/apmckinlay/gsuneido/runtime"
	"github.com/apmckinlay/gsuneido/util/ascii"
	"github.com/apmckinlay/gsuneido/util/str"
)

type DbmsClient struct {
	*csio.ReadWrite
	conn net.Conn
}

// helloSize is the size of the initial connection message from the server
// the size must match cSuneido and jSuneido
const helloSize = 50

func NewDbmsClient(addr string) *DbmsClient {
	conn, err := net.Dial("tcp", addr)
	if err != nil || !checkHello(conn) {
		panic("can't connect to " + addr + " " + err.Error())
	}
	return &DbmsClient{ReadWrite: csio.NewReadWrite(conn), conn: conn}
}

func checkHello(conn net.Conn) bool {
	var buf [helloSize]byte
	n, err := io.ReadFull(conn, buf[:])
	if n != helloSize || err != nil {
		return false
	}
	s := string(buf[:])
	if !strings.HasPrefix(s, "Suneido ") {
		return false
	}
	//TODO built date check
	return true
}

// Dbms interface

var _ IDbms = (*DbmsClient)(nil)

func (dc *DbmsClient) Admin(request string) {
	dc.PutCmd(commands.Admin).PutStr(request).Request()
}

func (dc *DbmsClient) Auth(s string) bool {
	if s == "" {
		return false
	}
	dc.PutCmd(commands.Auth).PutStr(s).Request()
	return dc.GetBool()
}

func (dc *DbmsClient) Check() string {
	dc.PutCmd(commands.Check).Request()
	return dc.GetStr()
}

func (dc *DbmsClient) Connections() Value {
	dc.PutCmd(commands.Connections).Request()
	ob := dc.GetVal().(*SuObject)
	ob.SetReadOnly()
	return ob
}

func (dc *DbmsClient) Cursors() int {
	dc.PutCmd(commands.Cursors).Request()
	return dc.GetInt()
}

func (dc *DbmsClient) Dump(table string) string {
	dc.PutCmd(commands.Dump).PutStr(table).Request()
	return dc.GetStr()
}

func (dc *DbmsClient) Exec(_ *Thread, args Value) Value {
	dc.PutCmd(commands.Exec).PutVal(args).Request()
	return dc.ValueResult()
}

func (dc *DbmsClient) Final() int {
	dc.PutCmd(commands.Final).Request()
	return dc.GetInt()
}

func (dc *DbmsClient) Get(tn int, query string, which byte) (Row, *Header) {
	dc.PutCmd(commands.Get1).PutByte(which).PutInt(tn).PutStr(query).Request()
	if !dc.GetBool() {
		return nil, nil
	}
	adr := dc.GetInt()
	hdr := dc.getHdr()
	row := dc.getRow(adr)
	return row, hdr
}

func (dc *DbmsClient) Info() Value {
	dc.PutCmd(commands.Info).Request()
	return dc.GetVal()
}

func (dc *DbmsClient) Kill(sessionid string) int {
	dc.PutCmd(commands.Kill).PutStr(sessionid).Request()
	return dc.GetInt()
}

func (dc *DbmsClient) Load(table string) int {
	dc.PutCmd(commands.Load).PutStr(table).Request()
	return dc.GetInt()
}

func (dc *DbmsClient) Log(s string) {
	dc.PutCmd(commands.Log).PutStr(s).Request()
}

func (dc *DbmsClient) LibGet(name string) []string {
	dc.PutCmd(commands.LibGet).PutStr(name).Request()
	n := dc.GetSize()
	v := make([]string, 2*n)
	sizes := make([]int, n)
	for i := 0; i < 2*n; i += 2 {
		v[i] = dc.GetStr() // library
		sizes[i/2] = dc.GetSize()
	}
	for i := 1; i < 2*n; i += 2 {
		v[i] = dc.GetN(sizes[i/2]) // text
	}
	return v
}

func (dc *DbmsClient) Libraries() *SuObject {
	dc.PutCmd(commands.Libraries).Request()
	return dc.getStrings()
}

func (dc *DbmsClient) getStrings() *SuObject {
	n := dc.GetInt()
	ob := NewSuObject()
	for ; n > 0; n-- {
		ob.Add(SuStr(dc.GetStr()))
	}
	return ob
}

func (dc *DbmsClient) Nonce() string {
	dc.PutCmd(commands.Nonce).Request()
	return dc.GetStr()
}

func (dc *DbmsClient) Run(code string) Value {
	dc.PutCmd(commands.Run).PutStr(code).Request()
	return dc.ValueResult()
}

func (dc *DbmsClient) SessionId(id string) string {
	dc.PutCmd(commands.SessionId).PutStr(id).Request()
	return dc.GetStr()
}

func (dc *DbmsClient) Size() int64 {
	dc.PutCmd(commands.Size).Request()
	return dc.GetInt64()
}

func (dc *DbmsClient) Timestamp() SuDate {
	dc.PutCmd(commands.Timestamp).Request()
	return dc.GetVal().(SuDate)
}

func (dc *DbmsClient) Token() string {
	dc.PutCmd(commands.Token).Request()
	return dc.GetStr()
}

func (dc *DbmsClient) Transaction(update bool) ITran {
	dc.PutCmd(commands.Transaction).PutBool(update).Request()
	tn := dc.GetInt()
	return &TranClient{dc: dc, tn: tn}
}

func (dc *DbmsClient) Transactions() *SuObject {
	dc.PutCmd(commands.Transactions).Request()
	ob := NewSuObject()
	for n := dc.GetInt(); n > 0; n-- {
		ob.Add(IntVal(dc.GetInt()))
	}
	return ob
}

func (dc *DbmsClient) Unuse(lib string) bool {
	panic("can't Unuse('" + lib + "')\n" +
		"When client-server, only the server can Unuse")
}

func (dc *DbmsClient) Use(lib string) bool {
	if _, ok := ContainerFind(dc.Libraries(), SuStr(lib)); ok {
		return false
	}
	panic("can't Use('" + lib + "')\n" +
		"When client-server, only the server can Use")
}

func (dc *DbmsClient) Close() {
	dc.conn.Close()
}

// ------------------------------------------------------------------

func (dc *DbmsClient) getHdr() *Header {
	n := dc.GetInt()
	fields := make([]string, 0, n)
	columns := make([]string, 0, n)
	for i := 0; i < n; i++ {
		s := dc.GetStr()
		if ascii.IsUpper(s[0]) {
			s = str.UnCapitalize(s)
		} else if !strings.HasSuffix(s, "_lower!") {
			fields = append(fields, s)
		}
		if s != "-" {
			columns = append(columns, s)
		}
	}
	return &Header{Fields: [][]string{fields}, Columns: columns}
}

func (dc *DbmsClient) getRow(adr int) Row {
	return Row([]DbRec{{Record(dc.GetStr()), adr}})
}

// ------------------------------------------------------------------

type TranClient struct {
	dc *DbmsClient
	tn int
}

var _ ITran = (*TranClient)(nil)

func (tc *TranClient) Abort() {
	tc.dc.PutCmd(commands.Abort).PutInt(tc.tn).Request()
}

func (tc *TranClient) Complete() string {
	tc.dc.PutCmd(commands.Commit).PutInt(tc.tn).Request()
	if tc.dc.GetBool() {
		return ""
	}
	return tc.dc.GetStr()
}

func (tc *TranClient) Erase(adr int) {
	tc.dc.PutCmd(commands.Erase).PutInt(tc.tn).PutInt(adr).Request()
}

func (tc *TranClient) Get(query string, which byte) (Row, *Header) {
	return tc.dc.Get(tc.tn, query, which)
}

func (tc *TranClient) Query(query string) IQuery {
	tc.dc.PutCmd(commands.Query).PutInt(tc.tn).PutStr(query).Request()
	qn := tc.dc.GetInt()
	return &QueryClient{dc: tc.dc, qn: qn}
}

func (tc *TranClient) Request(request string) int {
	tc.dc.PutCmd(commands.Request).PutInt(tc.tn).PutStr(request).Request()
	return tc.dc.GetInt()
}

func (tc *TranClient) Update(adr int, rec Record) int {
	tc.dc.PutCmd(commands.Update).
		PutInt(tc.tn).PutInt(adr).PutStr(string(rec)).Request()
	return tc.dc.GetInt()
}

func (tc *TranClient) String() string {
	return "Transaction" + strconv.Itoa(tc.tn)
}

// ------------------------------------------------------------------

type QueryClient struct {
	dc   *DbmsClient
	qn   int
	hdr  *Header
	keys *SuObject // cache
}

var _ IQuery = (*QueryClient)(nil)

func (qc *QueryClient) Close() {
	qc.dc.PutCmd(commands.Close).PutInt(qc.qn).PutByte('q').Request()
}

func (qc *QueryClient) Get(dir Dir) Row {
	qc.dc.PutCmd(commands.Get).
		PutByte(byte(dir)).PutInt(0).PutInt(qc.qn).Request()
	if !qc.dc.GetBool() {
		return nil
	}
	adr := qc.dc.GetInt()
	row := qc.dc.getRow(adr)
	return row
}

func (qc *QueryClient) Header() *Header {
	if qc.hdr == nil {
		qc.dc.PutCmd(commands.Header).PutInt(qc.qn).PutByte('q').Request()
		qc.hdr = qc.dc.getHdr()
	}
	return qc.hdr
}

func (qc *QueryClient) Keys() *SuObject {
	if qc.keys == nil {
		qc.dc.PutCmd(commands.Keys).PutInt(qc.qn).PutByte('q').Request()
		qc.keys = NewSuObject()
		nk := qc.dc.GetInt()
		for ; nk > 0; nk-- {
			cb := str.CommaBuilder{}
			n := qc.dc.GetInt()
			for ; n > 0; n-- {
				cb.Add(qc.dc.GetStr())
			}
			qc.keys.Add(SuStr(cb.String()))
		}
	}
	return qc.keys
}

func (qc *QueryClient) Order() *SuObject {
	qc.dc.PutCmd(commands.Order).PutInt(qc.qn).PutByte('q').Request()
	return qc.dc.getStrings()
}

func (qc *QueryClient) Output(rec Record) {
	qc.dc.PutCmd(commands.Output).PutInt(qc.qn).PutStr(string(rec)).Request()
}

func (qc *QueryClient) Rewind() {
	qc.dc.PutCmd(commands.Rewind).PutInt(qc.qn).PutByte('q').Request()
}

func (qc *QueryClient) Strategy() string {
	qc.dc.PutCmd(commands.Strategy).PutInt(qc.qn).PutByte('q').Request()
	return qc.dc.GetStr()
}
