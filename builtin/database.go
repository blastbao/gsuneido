// Copyright Suneido Software Corp. All rights reserved.
// Governed by the MIT license found in the LICENSE file.

package builtin

import (
	. "github.com/apmckinlay/gsuneido/runtime"
)

type suDatabaseGlobal struct {
	SuBuiltin
}

func init() {
	name, ps := paramSplit("Database(string)")
	Global.Builtin(name, &suDatabaseGlobal{
		SuBuiltin{Fn: databaseCallClass,
			BuiltinParams: BuiltinParams{ParamSpec: *ps}}})
}

func databaseCallClass(t *Thread, args []Value) Value {
	t.Dbms().Admin(ToStr(args[0]))
	return nil
}

var databaseMethods = Methods{
	"Auth": method("(data)", func(t *Thread, this Value, args []Value) Value {
		return SuBool(t.Dbms().Auth(ToStr(args[0])))
	}),
	"Check": method("()", func(t *Thread, this Value, args []Value) Value {
		return SuStr(t.Dbms().Check())
	}),
	"Connections": method("()", func(t *Thread, this Value, args []Value) Value {
		return t.Dbms().Connections()
	}),
	"CurrentSize": method("()", func(t *Thread, this Value, args []Value) Value {
		return IntVal(int(t.Dbms().Size()))
	}),
	"Cursors": method("()", func(t *Thread, this Value, args []Value) Value {
		return IntVal(t.Dbms().Cursors())
	}),
	"Dump": method("(table = '')", func(t *Thread, this Value, args []Value) Value {
		return SuStr(t.Dbms().Dump(ToStr(args[0])))
	}),
	"Final": method("()", func(t *Thread, this Value, args []Value) Value {
		return IntVal(t.Dbms().Final())
	}),
	"Info": method("()", func(t *Thread, this Value, args []Value) Value {
		return t.Dbms().Info()
	}),
	"Kill": method("(sessionId)", func(t *Thread, this Value, args []Value) Value {
		return IntVal(t.Dbms().Kill(ToStr(args[0])))
	}),
	"Load": method("(table)", func(t *Thread, this Value, args []Value) Value {
		return IntVal(t.Dbms().Load(ToStr(args[0])))
	}),
	"Nonce": method("()", func(t *Thread, this Value, args []Value) Value {
		return SuStr(t.Dbms().Nonce())
	}),
	"SessionId": method("(id = '')", func(t *Thread, this Value, args []Value) Value {
		return SuStr(t.Dbms().SessionId(ToStr(args[0])))
	}),
	"TempDest": method0(func(Value) Value {
		return Zero
	}),
	"Token": method("()", func(t *Thread, this Value, args []Value) Value {
		return SuStr(t.Dbms().Token())
	}),
	"Transactions": method("()", func(t *Thread, this Value, args []Value) Value {
		return t.Dbms().Transactions()
	}),
}

func (d *suDatabaseGlobal) Lookup(t *Thread, method string) Callable {
	if f, ok := databaseMethods[method]; ok {
		return f
	}
	return d.SuBuiltin.Lookup(t, method) // for Params
}

func (d *suDatabaseGlobal) String() string {
	return "Database /* builtin class */"
}

var _ = builtin("DoWithoutTriggers(tables, block)",
	func(t *Thread, args []Value) Value {
		dbms := t.Dbms()
		ob := ToContainer(args[0])
		for i := ob.ListSize() - 1; i >= 0; i-- {
			table := ToStr(ob.ListGet(i))
			dbms.DisableTrigger(table)
			defer dbms.EnableTrigger(table)
		}
		return t.Call(args[1])
	})
