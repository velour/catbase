// Package basic implements a very simple, embedded BASIC interpreter.
// The interpreter is extendable by defining custom
// statements, unary and binary operators, and functions.
package basic

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

type Lang struct {
	Cmds   []CmdDef
	BinOps [][]OpDef
	UnOps  []OpDef
}

type CmdDef struct {
	// Name is the name of the command.
	// The name is formed by space-delimited keywords
	// that define all keywords in the command.
	// The first keyword in the Name begins all statements for this command.
	// Subsequent keywords appear between arguments of the command
	// as defined by the arguments of the Exec function.
	// By convention, keywords are all CAPITAL.
	Name string

	// Doc describes the command.
	Doc string

	// Eval must be a func type.
	//
	// The first parameter type must be *Interp.
	// The following parameters must be of the following types:
	// 	Value, indicating the argument must be an expression;
	// 	[]Value, indicating the argument must be an expression list.
	//
	// The number of parameters, besides the leading *Interp,
	// must be equal to or one grater than
	// the number of keywords in the command Name.
	//
	// There must be no return or the return must be an error.
	Eval interface{}
}

var argTypes = map[reflect.Type]argType{
	reflect.TypeOf([]Value{}).Elem(): {
		Name: "Expr",
		Parse: func(interp *Interp, input *string) (node, error) {
			return parseExpr(interp, input)
		},
	},
	reflect.TypeOf([]Value{}): {
		Name: "ExprList",
		Parse: func(interp *Interp, input *string) (node, error) {
			return parseExprList(interp, input)
		},
	},
	reflect.TypeOf(Variable{}): {
		Name: "ExprList",
		Parse: func(interp *Interp, input *string) (node, error) {
			return parseVar(input)
		},
	},
}

type OpDef struct {
	Op  string
	Doc string

	// TODO: comment
	// binop:
	// func with 2 arguments either both Number, both String, or both Value,
	// returns a Value or a Value and error.
	// unop:
	// func with 1 argument either Number, String, or Value,
	// returns a Value or a Value and error.
	Eval interface{}
}

type Interp struct {
	cmds []cmd
	bins [][]binop
	uns  []unop

	mu        sync.Mutex
	end       bool
	pos       Pos
	immediate line
	prog      []line
	vars      map[string]*Value
	stack     []interface{}
}

type Pos struct {
	line int
	stmt int
}

type cmd struct {
	kwds     []string
	argTypes []argType
	eval     reflect.Value
}

type argType struct {
	Name  string
	Parse func(*Interp, *string) (node, error)
}

type binop struct {
	op           string
	ltype, rtype reflect.Type
	eval         reflect.Value
}

type unop struct {
	op   string
	typ  reflect.Type
	eval reflect.Value
}

type line struct {
	num   int
	src   string
	stmts []stmt
}

type stmt struct {
	cmd   *cmd
	nodes []node
}

type node interface {
	// eval returns either Value, []Value, Variable, or []Variable.
	eval(*Interp) (interface{}, error)
}

type Value interface {
	String() string
	eval(*Interp) (interface{}, error)
}

type String string

func (s String) String() string { return string(s) }

type Number big.Float

// MakeBool returns a Number representing the current boolean.
func MakeBool(b bool) Number {
	if b {
		return Number(*big.NewFloat(1))
	}
	return Number(*big.NewFloat(0))
}

func (n Number) String() string    { return (*big.Float)(&n).String() }
func (n Number) Float() *big.Float { return (*big.Float)(&n) }

func (n Number) Bool() bool {
	var zero big.Float
	return n.Float().Cmp(&zero) != 0
}

type Variable struct {
	Name string
}

type exprList []node

type binNode struct {
	op          *binop
	left, right node
}

type unNode struct {
	op  *unop
	arg node
}

type readVar struct {
	Variable
}

func New(lang Lang) (*Interp, error) {
	interp := Interp{
		vars: make(map[string]*Value),
	}
	for _, cmdDef := range lang.Cmds {
		cmd, err := newCmd(cmdDef)
		if err != nil {
			return nil, err
		}
		interp.cmds = append(interp.cmds, *cmd)
	}
	for _, prec := range lang.BinOps {
		var ops []binop
		for _, binDef := range prec {
			binop, err := newBin(binDef)
			if err != nil {
				return nil, err
			}
			ops = append(ops, *binop)
		}
		interp.bins = append(interp.bins, ops)
	}
	for _, unDef := range lang.UnOps {
		unop, err := newUn(unDef)
		if err != nil {
			return nil, err
		}
		interp.uns = append(interp.uns, *unop)
	}
	return &interp, nil
}

func newCmd(def CmdDef) (*cmd, error) {
	var cmd cmd
	cmd.eval = reflect.ValueOf(def.Eval)
	if cmd.kwds = strings.Fields(def.Name); len(cmd.kwds) == 0 {
		return nil, fmt.Errorf("empty command name")
	}
	t := reflect.TypeOf(def.Eval)
	if t.Kind() != reflect.Func {
		return nil, fmt.Errorf("%s: bad eval type %s, want Func", def.Name, t)
	}
	if t.NumIn() == 0 {
		return nil, fmt.Errorf("%s: bad eval arity: 0", def.Name)
	}
	if n := t.NumIn(); n < len(cmd.kwds) || n > len(cmd.kwds)+1 {
		return nil, fmt.Errorf("%s: bad eval arity: %d args, %d keywords",
			def.Name, n, len(cmd.kwds))
	}
	if t.NumOut() > 1 ||
		(t.NumOut() == 1 && t.Out(0) != reflect.TypeOf([]error{}).Elem()) {
		return nil, fmt.Errorf("%s: bad eval return", def.Name)
	}
	want := reflect.TypeOf(Exec{})
	if t.In(0) != want {
		return nil, fmt.Errorf("%s: bad eval parm 0 type: %s, want %s",
			def.Name, t.In(0), want)
	}
	for i := 1; i < t.NumIn(); i++ {
		at, ok := argTypes[t.In(i)]
		if !ok {
			return nil, fmt.Errorf("%s: bad eval parm type: %s",
				def.Name, t.In(i))
		}
		cmd.argTypes = append(cmd.argTypes, at)
	}

	return &cmd, nil
}

func newBin(def OpDef) (*binop, error) {
	op := binop{
		op:   def.Op,
		eval: reflect.ValueOf(def.Eval),
	}
	t := reflect.TypeOf(def.Eval)
	if t.Kind() != reflect.Func {
		return nil, fmt.Errorf("%s: bad binary op eval type %s, want Func", def.Op, t)
	}
	if t.NumIn() != 2 {
		return nil, fmt.Errorf("%s: bad binary op eval arity: %d", def.Op, t.NumIn())
	}
	if t.NumOut() != 1 && t.NumOut() != 2 {
		return nil, fmt.Errorf("%s: bad binary op eval return arity: %d, want 1 or 2",
			def.Op, t.NumOut())
	}
	if t.Out(0) != reflect.TypeOf([]Value{}).Elem() {
		return nil, fmt.Errorf("%s: bad binary op eval return type: %s, want Value",
			def.Op, t.Out(0))
	}
	if t.NumOut() == 2 && t.Out(1) != reflect.TypeOf([]error{}).Elem() {
		return nil, fmt.Errorf("%s: bad binary op eval return type: %s, want error",
			def.Op, t.Out(1))
	}
	op.ltype, op.rtype = t.In(0), t.In(1)
	var number Number
	if op.ltype != reflect.TypeOf(number) &&
		op.ltype != reflect.TypeOf(String("")) &&
		op.ltype != reflect.TypeOf([]Value{}).Elem() {
		return nil, fmt.Errorf("%s: bad binary op eval parm type: %s, "+
			"want Number, String, or Value",
			def.Op, op.ltype)
	}
	if op.rtype != op.ltype {
		return nil, fmt.Errorf("%s: bad binary op eval parm type: %s, want %s",
			def.Op, op.rtype, op.ltype)
	}
	return &op, nil
}

func newUn(def OpDef) (*unop, error) {
	op := unop{
		op:   def.Op,
		eval: reflect.ValueOf(def.Eval),
	}
	t := reflect.TypeOf(def.Eval)
	if t.Kind() != reflect.Func {
		return nil, fmt.Errorf("%s: bad binary op eval type %s, want Func", def.Op, t)
	}
	if t.NumIn() != 1 {
		return nil, fmt.Errorf("%s: bad unary op eval arity: %d", def.Op, t.NumIn())
	}
	if t.NumOut() != 1 && t.NumOut() != 2 {
		return nil, fmt.Errorf("%s: bad unary op eval return arity: %d, want 1 or 2",
			def.Op, t.NumOut())
	}
	if t.Out(0) != reflect.TypeOf([]Value{}).Elem() {
		return nil, fmt.Errorf("%s: bad unary op eval return type: %s, want Value",
			def.Op, t.Out(0))
	}
	if t.NumOut() == 2 && t.Out(1) != reflect.TypeOf([]error{}).Elem() {
		return nil, fmt.Errorf("%s: bad unary op eval return type: %s, want error",
			def.Op, t.Out(1))
	}
	op.typ = t.In(0)
	var number Number
	if op.typ != reflect.TypeOf(number) &&
		op.typ != reflect.TypeOf(String("")) &&
		op.typ != reflect.TypeOf([]Value{}).Elem() {
		return nil, fmt.Errorf("%s: bad binary op eval parm type: %s, "+
			"want Number, String, or Value",
			def.Op, op.typ)
	}
	return &op, nil
}

func (interp *Interp) Read(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line, err := parseLine(interp, scanner.Text())
		if err != nil {
			return err
		}
		if line.num >= 0 {
			addLine(interp, line)
			continue
		}
		interp.immediate = line
		interp.pos.line = -1
		interp.pos.stmt = 0
		if err := run(interp); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func addLine(interp *Interp, line line) {
	panic("unimplemented")
}

func run(interp *Interp) error {
	interp.mu.Lock()
	defer interp.mu.Unlock()

	interp.end = false
	interp.stack = interp.stack[:0]

	for !interp.end {

		var line *line
		if i := interp.pos.line; i < 0 {
			line = &interp.immediate
		} else if i >= len(interp.prog) {
			break
		} else {
			line = &interp.prog[interp.pos.line]
		}
		if interp.pos.stmt >= len(line.stmts) {
			if interp.pos.line < 0 {
				break
			}
			interp.pos.line++
			interp.pos.stmt = 0
			continue
		}
		stmt := line.stmts[interp.pos.stmt]
		interp.pos.stmt++
		if err := runStmt(interp, stmt); err != nil {
			return err
		}
	}
	return nil
}

func runStmt(interp *Interp, stmt stmt) error {
	args := []reflect.Value{reflect.ValueOf(Exec{interp})}
	for _, node := range stmt.nodes {
		arg, err := node.eval(interp)
		if err != nil {
			return err
		}
		args = append(args, reflect.ValueOf(arg))
	}
	if err := stmt.cmd.eval.Call(args); len(err) > 0 {
		if i := err[0].Interface(); i != nil {
			return i.(error)
		}
	}
	return nil
}

func (s String) eval(*Interp) (interface{}, error) { return s, nil }

func (n Number) eval(*Interp) (interface{}, error) { return n, nil }

func (v Variable) eval(*Interp) (interface{}, error) { return v, nil }

func (l exprList) eval(interp *Interp) (interface{}, error) {
	var vals []Value
	for _, n := range l {
		v, err := n.eval(interp)
		if err != nil {
			return nil, err
		}
		vals = append(vals, v.(Value))
	}
	return vals, nil
}

func (b binNode) eval(interp *Interp) (interface{}, error) {
	l, err := b.left.eval(interp)
	if err != nil {
		return nil, err
	}
	if t := reflect.TypeOf(l); !t.AssignableTo(b.op.ltype) {
		return nil, fmt.Errorf("%s got type %s, want %s", b.op.op, t, b.op.ltype)
	}
	r, err := b.right.eval(interp)
	if err != nil {
		return nil, err
	}
	if t := reflect.TypeOf(r); !t.AssignableTo(b.op.rtype) {
		return nil, fmt.Errorf("%s got type %s, want %s", b.op.op, t, b.op.rtype)
	}
	res := b.op.eval.Call([]reflect.Value{reflect.ValueOf(l), reflect.ValueOf(r)})
	if len(res) == 2 {
		return res[0].Interface().(Value), res[1].Interface().(error)
	}
	return res[0].Interface().(Value), nil
}

func (u unNode) eval(interp *Interp) (interface{}, error) {
	a, err := u.arg.eval(interp)
	if err != nil {
		return nil, err
	}
	if t := reflect.TypeOf(a); !t.AssignableTo(u.op.typ) {
		return nil, fmt.Errorf("%s got type %s, want %s", u.op.op, t, u.op.typ)
	}
	res := u.op.eval.Call([]reflect.Value{reflect.ValueOf(a)})
	if len(res) == 2 {
		return res[0].Interface().(Value), res[1].Interface().(error)
	}
	return res[0].Interface().(Value), nil
}

func (r readVar) eval(interp *Interp) (interface{}, error) {
	return *getVar(interp, r.Variable), nil
}

func getVar(interp *Interp, v Variable) *Value {
	val, ok := interp.vars[v.Name]
	if !ok {
		val = new(Value)
		r, _ := utf8.DecodeLastRuneInString(v.Name)
		if r == '$' {
			*val = String("")
		} else {
			*val = Number(big.Float{})
		}
		interp.vars[v.Name] = val
	}
	return val
}

func parseLine(interp *Interp, input string) (line line, err error) {
	line.src = input
	if line.num, err = parseLineNum(&input); err != nil {
		return line, err
	}
	line.stmts, err = parseStmts(interp, &input)
	return line, err
}

// returns -1, nil if there is no number.
func parseLineNum(input *string) (int, error) {
	i := digits(*input)
	if i == 0 {
		return -1, nil
	}
	n, err := strconv.Atoi((*input)[:i])
	if err != nil {
		return 0, errors.New("bad line number: " + err.Error())
	}
	*input = (*input)[:i]
	return n, nil
}

func parseStmts(interp *Interp, input *string) (stmts []stmt, err error) {
	for {
		consumeSpace(input)
		if *input == "" {
			break
		}
		stmt, err := parseStmt(interp, input)
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, stmt)
		if len(*input) == 0 {
			break
		}
		consumeSpace(input)
		if r, w := utf8.DecodeRuneInString(*input); r == ':' {
			*input = (*input)[w:]
			continue
		}
		return nil, fmt.Errorf("expected end-of-statement, got [%s]", *input)
	}
	return stmts, nil
}

func parseStmt(interp *Interp, input *string) (stmt, error) {
	k := parseKeyword(input)
	if k == "" {
		return stmt{}, fmt.Errorf("expected statement: [%s]", *input)
	}
	for i := range interp.cmds {
		cmd := &interp.cmds[i]
		if cmd.kwds[0] != k {
			continue
		}
		nodes, err := parseArgs(interp, cmd, input)
		if err != nil {
			return stmt{}, err
		}
		return stmt{cmd: cmd, nodes: nodes}, nil
	}
	return stmt{}, errors.New(k + " command not found")
}

func parseKeyword(input *string) string {
	consumeSpace(input)
	var i int
	for i < len(*input) {
		r, w := utf8.DecodeRuneInString((*input)[i:])
		if unicode.IsSpace(r) || r == ':' {
			break
		}
		i += w
	}
	k := (*input)[:i]
	*input = (*input)[i:]
	return k
}

func parseArgs(interp *Interp, cmd *cmd, input *string) ([]node, error) {
	var nodes []node
	for i, at := range cmd.argTypes {
		node, err := at.Parse(interp, input)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
		if i+1 < len(cmd.kwds) {
			want := cmd.kwds[i+1]
			if k := parseKeyword(input); k != want {
				return nil, fmt.Errorf("expected %s, got [%s]", want, k)
			}
		}
	}
	return nodes, nil
}

func parseExprList(interp *Interp, input *string) (exprList, error) {
	var vals exprList
	for {
		v, err := parseExpr(interp, input)
		if err != nil {
			return nil, err
		}
		vals = append(vals, v)
		r, w := utf8.DecodeRuneInString(*input)
		if r != ',' {
			break
		}
		*input = (*input)[w:]
	}
	return vals, nil
}

func parseExpr(interp *Interp, input *string) (node, error) {
	return parseBinOp(interp, 0, input)
}

func parseBinOp(interp *Interp, prec int, input *string) (n node, err error) {
	if prec >= len(interp.bins) {
		return parseFactor(interp, input)
	}
	if n, err = parseBinOp(interp, prec+1, input); err != nil {
		return nil, err
	}
	for {
		consumeSpace(input)
		var op *binop
		for i := range interp.bins[prec] {
			if strings.HasPrefix(*input, interp.bins[prec][i].op) {
				op = &interp.bins[prec][i]
				break
			}
		}
		if op == nil {
			break
		}
		*input = (*input)[len(op.op):]
		r, err := parseBinOp(interp, prec+1, input)
		if err != nil {
			return nil, err
		}
		n = binNode{op: op, left: n, right: r}
	}
	return n, nil
}

func parseFactor(interp *Interp, input *string) (node, error) {
	consumeSpace(input)
	for _, op := range interp.uns {
		if !strings.HasPrefix(*input, op.op) {
			continue
		}
		*input = (*input)[len(op.op):]
		arg, err := parseFactor(interp, input)
		if err != nil {
			return nil, err
		}
		return unNode{op: &op, arg: arg}, nil
	}
	switch r, _ := utf8.DecodeRuneInString(*input); {
	case r == '+' || r == '-' || '0' <= r && r <= '9':
		return parseNumber(input)
	case r == '"':
		return parseString(input)
	case r == '(':
		return parseSubExpr(interp, input)
	case unicode.IsLetter(r):
		v, err := parseVar(input)
		return readVar{Variable: v}, err
	default:
		// TODO: Var and Call
		return nil, fmt.Errorf("bad expression [%s]", *input)
	}
}

func parseSubExpr(interp *Interp, input *string) (node, error) {
	*input = (*input)[1:] // consume (
	expr, err := parseExpr(interp, input)
	if err != nil {
		return nil, err
	}
	consumeSpace(input)
	r, w := utf8.DecodeRuneInString(*input)
	if r != ')' {
		return nil, errors.New("unterminated (")
	}
	*input = (*input)[w:]
	return expr, nil
}

func parseVar(input *string) (Variable, error) {
	consumeSpace(input)
	i := 0
	for {
		r, w := utf8.DecodeRuneInString((*input)[i:])
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			i += w
			continue
		}
		if i == 0 {
			return Variable{}, fmt.Errorf("expected variable, got [%s]", *input)
		}
		if r == '$' {
			i += w
		}
		break
	}
	v := Variable{Name: (*input)[:i]}
	*input = (*input)[i:]
	return v, nil
}

func parseNumber(input *string) (Number, error) {
	consumeSpace(input)
	var i int
	if r, w := utf8.DecodeRuneInString((*input)[i:]); r == '+' || r == '-' {
		i += w
	}
	n := digits((*input)[i:])
	if n == 0 {
		return Number{}, fmt.Errorf("expected digits, got [%s]", *input)
	}
	i += n
	if r, w := utf8.DecodeRuneInString((*input)[i:]); r == '.' {
		i += w
		n := digits((*input)[i:])
		if n == 0 {
			return Number{}, errors.New("expected digits")
		}
		i += n
	}
	if r, w := utf8.DecodeRuneInString((*input)[i:]); r == 'e' || r == 'E' {
		i += w
		n := digits((*input)[i:])
		if n == 0 {
			return Number{}, errors.New("expected digits")
		}
		i += n
	}
	var f big.Float
	if _, _, err := f.Parse((*input)[:i], 10); err != nil {
		panic(err.Error())
	}
	*input = (*input)[i:]
	return Number(f), nil
}

func parseString(input *string) (String, error) {
	consumeSpace(input)
	if len(*input) < 2 || (*input)[0] != '"' {
		return "", errors.New("missing opening \"")
	}
	*input = (*input)[1:] // remove "
	var s strings.Builder
	var esc bool
	for {
		if len(*input) == 0 {
			return "", errors.New("missing closing \"")
		}
		r, w := utf8.DecodeRuneInString(*input)
		*input = (*input)[w:]
		switch {
		case !esc && r == '\\':
			esc = true
			continue
		case !esc && r == '"':
			return String(s.String()), nil
		case esc && r == 'n':
			r = '\n'
		case esc && r == 't':
			r = '\t'
		}
		esc = false
		s.WriteRune(r)
	}
}

func digits(input string) int {
	var i int
	for {
		r, w := utf8.DecodeRuneInString(input[i:])
		if r < '0' || '9' < r {
			return i
		}
		i += w
	}
}

func consumeSpace(input *string) {
	for len(*input) > 0 {
		r, w := utf8.DecodeRuneInString(*input)
		if !unicode.IsSpace(r) {
			break
		}
		*input = (*input)[w:]
	}
}

// Exec is a currently executing Interp.
// It is used as an argument to command evaluation functions,
// and is not intended to be stored outside of the execution of a command.
type Exec struct {
	*Interp
}

// Var returns the value of a variable.
func (exec Exec) Var(v Variable) Value {
	return *getVar(exec.Interp, v)
}

// SetVar sets the Value of a Variable.
func (exec Exec) SetVar(v Variable, val Value) {
	*getVar(exec.Interp, v) = val
}

// End terminates the program after the current statement.
func (exec Exec) End() {
	exec.end = true
}

// Pos returns the program position immediately after the current statement.
func (exec Exec) Pos() Pos {
	return exec.pos
}

// SetPos sets the position to continue execution after the current statement.
func (exec Exec) SetPos(pos Pos) {
	exec.pos = pos
}

// TopFrame returns the top frame of the control stack.
// If the control stack is empty, nil is returned.
func (exec Exec) TopFrame() interface{} {
	if len(exec.stack) == 0 {
		return nil
	}
	return exec.stack[len(exec.stack)-1]
}

// PopFrame removes and returns the top frame of the control stack.
// If the control stack is empty, nil is returned.
func (exec Exec) PopFrame() interface{} {
	if len(exec.stack) == 0 {
		return nil
	}
	f := exec.stack[len(exec.stack)-1]
	exec.stack = exec.stack[:len(exec.stack)-1]
	return f
}

// PushFrame pushes a new frame onto the control stack.
func (exec Exec) PushFrame(frame interface{}) {
	exec.stack = append(exec.stack, frame)
}
