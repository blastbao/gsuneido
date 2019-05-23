package builtin

import (
	. "github.com/apmckinlay/gsuneido/runtime"
)

type SuDatabaseGlobal struct {
	SuBuiltin
}

func init() {
	name, ps := paramSplit("Database(string)")
	Global.Builtin(name, &SuDatabaseGlobal{
		SuBuiltin{databaseCallClass, BuiltinParams{ParamSpec: *ps}}})
}

func databaseCallClass(t *Thread, args ...Value) Value {
	t.Dbms().Admin(IfStr(args[0]))
	return nil
}

var databaseMethods = Methods{
	"Auth": method("(data)", func(t *Thread, this Value, args ...Value) Value {
		return SuBool(t.Dbms().Auth(IfStr(args[0])))
	}),
	"Check": method("()", func(t *Thread, this Value, args ...Value) Value {
		return SuStr(t.Dbms().Check())
	}),
	"Connections": method("()", func(t *Thread, this Value, args ...Value) Value {
		return t.Dbms().Connections()
	}),
	"CurrentSize": method("()", func(t *Thread, this Value, args ...Value) Value {
		return IntVal(int(t.Dbms().Size()))
	}),
	"Cursors": method("()", func(t *Thread, this Value, args ...Value) Value {
		return IntVal(t.Dbms().Cursors())
	}),
	"Dump": method("(table = false)", func(t *Thread, this Value, args ...Value) Value {
		return SuStr(t.Dbms().Dump(IfStr(args[0])))
	}),
	"Final": method("()", func(t *Thread, this Value, args ...Value) Value {
		return IntVal(t.Dbms().Final())
	}),
	"Info": method("()", func(t *Thread, this Value, args ...Value) Value {
		return t.Dbms().Info()
	}),
	"Kill": method("(sessionId)", func(t *Thread, this Value, args ...Value) Value {
		return IntVal(t.Dbms().Kill(IfStr(args[0])))
	}),
	"Load": method("(table = false)", func(t *Thread, this Value, args ...Value) Value {
		return IntVal(t.Dbms().Load(IfStr(args[0])))
	}),
	"Nonce": method("()", func(t *Thread, this Value, args ...Value) Value {
		return SuStr(t.Dbms().Nonce())
	}),
	"SessionId": method("(id = '')", func(t *Thread, this Value, args ...Value) Value {
		return SuStr(t.Dbms().SessionId(IfStr(args[0])))
	}),
	"TempDest": method0(func(Value) Value {
		return Zero
	}),
	"Token": method("()", func(t *Thread, this Value, args ...Value) Value {
		return SuStr(t.Dbms().Token())
	}),
	"Transactions": method("()", func(t *Thread, this Value, args ...Value) Value {
		return t.Dbms().Transactions()
	}),
}

func (d *SuDatabaseGlobal) Lookup(t *Thread, method string) Callable {
	if f, ok := databaseMethods[method]; ok {
		return f
	}
	return d.Lookup(t, method) // for Params
}

func (d *SuDatabaseGlobal) String() string {
	return "Database /* builtin class */"
}