package op

// NOTE: opcodes must match disasm.go

const (
	RETURN = iota
	POP
	DUP
	DUP2
	DUPX2
	INT
	VALUE
	IS
	ISNT
	MATCH
	MATCHNOT
	LT
	LTE
	GT
	GTE
	ADD
	SUB
	CAT
	MUL
	DIV
	MOD
	LSHIFT
	RSHIFT
	BITOR
	BITAND
	BITXOR
	BITNOT
	NOT
	UPLUS
	UMINUS
	LOAD
	STORE
	DYLOAD
	GET
	PUT
	GLOBAL
	TRUE
	FALSE
	ZERO
	ONE
	MAXINT
	EMPTYSTR
	OR
	AND
	BOOL
	Q_MARK
	IN
	JUMP
	TJUMP
	FJUMP
	EQJUMP
	NEJUMP
	THROW
	TRY
	RANGETO
	RANGELEN
	THIS
	CALLFUNC
	CALLFUNC0
	CALLFUNC1
	CALLFUNC2
	CALLFUNC3
	CALLFUNC4
	CALLMETH
	CALLMETH0
	CALLMETH1
	CALLMETH2
	CALLMETH3
)
