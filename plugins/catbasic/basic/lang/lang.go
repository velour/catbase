package lang

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strings"

	"github.com/velour/catbase/plugins/catbasic/basic"
)

var Default = basic.Lang{
	Cmds: []basic.CmdDef{
		{
			Name: "END",
			Doc:  "ends execution of the program",
			Eval: endCmd,
		},
		{
			Name: "PRINT",
			Doc:  "prints each value, separated by a space, then a newline",
			Eval: printCmd,
		},
		{
			Name: "LET =",
			Doc:  "assigns a value to a variable",
			Eval: letCmd,
		},
		{
			Name: "FOR = TO",
			Doc:  "begins a for loop. The loop the variable is assigned and execution continues at the following statement. A subsequent NEXT statement with a variable matching the FOR variable will increment the variable; if the value is less than the TO value, then executinon will continue from the statement following the FOR statement, otherwise from the statement following the NEXT statement",
			Eval: forCmd,
		},
		{
			Name: "NEXT",
			Doc:  "closes the body of a FOR loop. See FOR for details",
			Eval: nextCmd,
		},
	},
	BinOps: [][]basic.OpDef{
		{
			{Op: "OR", Eval: logic(func(a, b bool) bool { return a || b })},
		},
		{
			{Op: "AND", Eval: logic(func(a, b bool) bool { return a && b })},
		},
		{
			{Op: "<", Eval: rel("<", func(c int) bool { return c < 0 })},
			{Op: "<=", Eval: rel("<=", func(c int) bool { return c <= 0 })},
			{Op: ">", Eval: rel(">", func(c int) bool { return c > 0 })},
			{Op: ">=", Eval: rel(">=", func(c int) bool { return c >= 0 })},
			{Op: "=", Eval: rel("=", func(c int) bool { return c == 0 })},
			{Op: "<>", Eval: rel("<>", func(c int) bool { return c != 0 })},
		},
		{
			{Op: "+", Eval: add},
			{Op: "-", Eval: arithmetic("-", (*big.Float).Sub)},
		},
		{
			{Op: "*", Eval: arithmetic("-", (*big.Float).Mul)},
			{Op: "/", Eval: arithmetic("-", (*big.Float).Quo)},
		},
	},
	UnOps: []basic.OpDef{
		{Op: "NOT", Eval: not},
	},
}

func endCmd(exec basic.Exec) {
	exec.End()
}

func printCmd(exec basic.Exec, values []basic.Value) {
	for i, val := range values {
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Print(val.String())
	}
	fmt.Print("\n")
}

func letCmd(exec basic.Exec, vr basic.Variable, val basic.Value) error {
	if t := reflect.TypeOf(exec.Var(vr)); t != reflect.TypeOf(val) {
		return fmt.Errorf("cannot assign %T to a variable of type %T", val, t)
	}
	exec.SetVar(vr, val)
	return nil
}

type forFrame struct {
	name string
	v    basic.Variable
	to   basic.Number
	pos  basic.Pos
}

func forCmd(exec basic.Exec, v basic.Variable, from, to basic.Value) error {
	if _, ok := exec.Var(v).(basic.Number); !ok {
		return fmt.Errorf("FOR variable %s is a %T, not a Number", v.Name, exec.Var(v))
	}
	f, ok := from.(basic.Number)
	if !ok {
		return fmt.Errorf("FOR start value is a %T, not a Number", from)
	}
	t, ok := to.(basic.Number)
	if !ok {
		return fmt.Errorf("FOR end value is a %T, not a Number", to)
	}
	exec.SetVar(v, f)
	exec.PushFrame(forFrame{
		name: v.Name,
		v:    v,
		to:   t,
		pos:  exec.Pos(),
	})
	return nil
}

func nextCmd(exec basic.Exec, v basic.Variable) error {
	top, ok := exec.TopFrame().(forFrame)
	if !ok {
		return errors.New("NEXT called not within a FOR loop")
	}
	if v.Name != top.name {
		return fmt.Errorf("interleaved FOR loops: %s and %s", v.Name, top.name)
	}
	f := exec.Var(top.v).(basic.Number).Float()
	exec.SetVar(top.v, basic.Number(*f.Add(f, big.NewFloat(1))))
	if f.Cmp(top.to.Float()) >= 0 {
		exec.PopFrame()
	} else {
		exec.SetPos(top.pos)
	}
	return nil
}

func logic(f func(a, b bool) bool) func(basic.Number, basic.Number) basic.Value {
	return func(a, b basic.Number) basic.Value {
		return basic.MakeBool(f(a.Bool(), b.Bool()))
	}
}

func rel(n string, ok func(int) bool) func(basic.Value, basic.Value) (basic.Value, error) {
	return func(l, r basic.Value) (basic.Value, error) {
		var c int
		switch l := l.(type) {
		case basic.String:
			if r, ok := r.(basic.String); !ok {
				return nil, fmt.Errorf("String %s expects String, got %T", n, r)
			} else {
				c = strings.Compare(string(l), string(r))
			}
		case basic.Number:
			if r, ok := r.(basic.Number); !ok {
				return nil, fmt.Errorf("Number %s expects Numbe, got %T", n, r)
			} else {
				c = l.Float().Cmp(r.Float())
			}
		default:
			return nil, fmt.Errorf("%s expects String or Number, got %T", n, l)
		}
		return basic.MakeBool(ok(c)), nil
	}
}

func arithmetic(n string, op func(z, a, b *big.Float) *big.Float) func(l, r basic.Number) basic.Value {
	return func(l, r basic.Number) basic.Value {
		var z big.Float
		op(&z, l.Float(), r.Float())
		return basic.Number(z)
	}
}

func add(l, r basic.Value) basic.Value {
	switch l := l.(type) {
	case basic.String:
		return basic.String(l + r.(basic.String))
	case basic.Number:
		return arithmetic("+", (*big.Float).Add)(l, r.(basic.Number))
	default:
		panic("impossible")
	}
}

func not(b basic.Number) basic.Value { return basic.MakeBool(!b.Bool()) }
