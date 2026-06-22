package graph

import (
	"fmt"
	"sort"
	"strings"

	"github.com/leftmike/research/cypher"
)

// evalFunction evaluates a scalar function call. Aggregating functions are not
// supported in this read-only executor and return an error.
func (ev *evaluator) evalFunction(x *cypher.FunctionCall, s scope) (Value, error) {
	if x.Distinct || x.Star {
		return nil, fmt.Errorf("graph: aggregation function %s is not supported", x.Name)
	}
	name := strings.ToLower(x.Name)

	args := make([]Value, len(x.Args))
	for i, a := range x.Args {
		v, err := ev.eval(a, s)
		if err != nil {
			return nil, err
		}
		args[i] = v
	}

	switch name {
	case "id":
		if err := wantArgs(name, args, 1); err != nil {
			return nil, err
		}
		switch e := args[0].(type) {
		case *Node:
			return e.ID, nil
		case *Relationship:
			return e.ID, nil
		case nil:
			return nil, nil
		default:
			return nil, fmt.Errorf("graph: id() requires a node or relationship")
		}

	case "labels":
		if err := wantArgs(name, args, 1); err != nil {
			return nil, err
		}
		n, ok := args[0].(*Node)
		if !ok {
			if args[0] == nil {
				return nil, nil
			}
			return nil, fmt.Errorf("graph: labels() requires a node")
		}
		out := make([]Value, len(n.Labels))
		for i, l := range n.Labels {
			out[i] = l
		}
		return out, nil

	case "type":
		if err := wantArgs(name, args, 1); err != nil {
			return nil, err
		}
		r, ok := args[0].(*Relationship)
		if !ok {
			if args[0] == nil {
				return nil, nil
			}
			return nil, fmt.Errorf("graph: type() requires a relationship")
		}
		return r.Type, nil

	case "keys":
		if err := wantArgs(name, args, 1); err != nil {
			return nil, err
		}
		return keysOf(args[0]), nil

	case "properties":
		if err := wantArgs(name, args, 1); err != nil {
			return nil, err
		}
		return propertiesOf(args[0]), nil

	case "toupper":
		return stringFn(name, args, strings.ToUpper)
	case "tolower":
		return stringFn(name, args, strings.ToLower)
	case "trim":
		return stringFn(name, args, strings.TrimSpace)

	case "length", "size":
		if err := wantArgs(name, args, 1); err != nil {
			return nil, err
		}
		switch v := args[0].(type) {
		case nil:
			return nil, nil
		case string:
			return int64(len([]rune(v))), nil
		case []Value:
			return int64(len(v)), nil
		default:
			return nil, fmt.Errorf("graph: %s() requires a string or list", name)
		}

	case "coalesce":
		for _, a := range args {
			if a != nil {
				return a, nil
			}
		}
		return nil, nil

	case "tostring":
		if err := wantArgs(name, args, 1); err != nil {
			return nil, err
		}
		if args[0] == nil {
			return nil, nil
		}
		return formatValue(args[0]), nil

	default:
		return nil, fmt.Errorf("graph: unsupported function %s()", x.Name)
	}
}

func wantArgs(name string, args []Value, n int) error {
	if len(args) != n {
		return fmt.Errorf("graph: %s() expects %d argument(s), got %d", name, n, len(args))
	}
	return nil
}

func stringFn(name string, args []Value, fn func(string) string) (Value, error) {
	if err := wantArgs(name, args, 1); err != nil {
		return nil, err
	}
	if args[0] == nil {
		return nil, nil
	}
	str, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("graph: %s() requires a string", name)
	}
	return fn(str), nil
}

func keysOf(v Value) Value {
	props := rawProps(v)
	if props == nil {
		if v == nil {
			return nil
		}
		return []Value{}
	}
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]Value, len(keys))
	for i, k := range keys {
		out[i] = k
	}
	return out
}

func propertiesOf(v Value) Value {
	props := rawProps(v)
	if props == nil {
		if v == nil {
			return nil
		}
		return map[string]Value{}
	}
	m := make(map[string]Value, len(props))
	for k, val := range props {
		m[k] = val
	}
	return m
}

// rawProps returns the underlying property map of a node, relationship or map.
func rawProps(v Value) map[string]Value {
	switch o := v.(type) {
	case *Node:
		return o.Props
	case *Relationship:
		return o.Props
	case map[string]Value:
		return o
	default:
		return nil
	}
}
