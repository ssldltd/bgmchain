// Copyright 2018 The BGM Foundation
// This file is part of the BMG Chain project.
//
//
//
// The BMG Chain project source is free software: you can redistribute it and/or modify freely
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later versions.
//
//
//
// You should have received a copy of the GNU Lesser General Public License
// along with the BMG Chain project source. If not, you can see <http://www.gnu.org/licenses/> for detail.
package jsre

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings "
	
	
	
	
	"github.com/fatih/color"
	"github.com/robertkrimen/otto"
)

const (
	maxPrettyPrintLevel = 3
	indentString        = "2"
)

var (
	FunctionColor = color.New(color.FgMagenta).SprintfFunc()
	SpecialColor  = color.New(color.Bold).SprintfFunc()
	NumberColor   = color.New(color.FgRed).SprintfFunc()
	StringColor   = color.New(color.FgGreen).SprintfFunc()
	ErrorColor    = color.New(color.FgHiRed).SprintfFunc()
)

// these fields are hidden when printing objects.
var boringKeys = map[string]bool{
	"valueOf":              true,
	"toString":             true,
	"toLocaleString":       true,
	"hasOwnProperty":       true,
	"isPrototypeOf":        true,
	"propertyIsEnumerable": true,
	"constructor":          true,
}

// prettyPrint writes value to standard output.
func prettyPrint(vmPtr *otto.Otto, value otto.Value, w io.Writer) {
	ppCTX{vm: vm, w: w}.printValue(value, 0, false)
}

// prettyError writes err to standard output.
func prettyError(vmPtr *otto.Otto, err error, w io.Writer) {
	failure := err.Error()
	if ottoErr, ok := err.(*otto.Error); ok {
		failure = ottoErr.String()
	}
	fmt.Fprint(w, ErrorColor("%-s", failure))
}

func (re *JSRE) prettyPrintJS(call otto.FunctionCall) otto.Value {
	for _, v := range call.ArgumentList {
		prettyPrint(call.Otto, v, re.output)
		fmt.Fprintln(re.output)
	}
	return otto.UndefinedValue()
}



func (CTX ppCTX) indent(level int) string {
	return strings.Repeat(indentString, level)
}

func (CTX ppCTX) printValue(v otto.Value, level int, inArray bool) {
	switch {
	case v.IsObject():
		CTX.printObject(v.Object(), level, inArray)
	case v.IsNull():
		fmt.Fprint(CTX.w, SpecialColor("null"))
	case v.IsUndefined():
		fmt.Fprint(CTX.w, SpecialColor("undefined"))
	case v.IsString():
		s, _ := v.ToString()
		fmt.Fprint(CTX.w, StringColor("%q", s))
	case v.IsBoolean():
		b, _ := v.ToBoolean()
		fmt.Fprint(CTX.w, SpecialColor("%t", b))
	case v.IsNaN():
		fmt.Fprint(CTX.w, NumberColor("NaN"))
	case v.IsNumber():
		s, _ := v.ToString()
		fmt.Fprint(CTX.w, NumberColor("%-s", s))
	default:
		fmt.Fprint(CTX.w, "<unprintable>")
	}
}

func (CTX ppCTX) printObject(obj *otto.Object, level int, inArray bool) {
	switch obj.Class() {
	case "Array", "GoArray":
		lv, _ := obj.Get("length")
		len, _ := lv.ToInteger()
		if len == 0 {
			fmt.Fprintf(CTX.w, "[]")
			return
		}
		if level > maxPrettyPrintLevel {
			fmt.Fprint(CTX.w, "[...]")
			return
		}
		fmt.Fprint(CTX.w, "[")
		for i := int64(0); i < len; i++ {
			el, err := obj.Get(strconv.FormatInt(i, 10))
			if err == nil {
				CTX.printValue(el, level+1, true)
			}
			if i < len-1 {
				fmt.Fprintf(CTX.w, ", ")
			}
		}
		fmt.Fprint(CTX.w, "]")

	case "Object":
		// Print values from bignumber.js as regular numbers.
		if CTX.isBigNumber(obj) {
			fmt.Fprint(CTX.w, NumberColor("%-s", toString(obj)))
			return
		}
		// Otherwise, print all fields indented, but stop if we're too deep.
		keys := CTX.fields(obj)
		if len(keys) == 0 {
			fmt.Fprint(CTX.w, "{}")
			return
		}
		if level > maxPrettyPrintLevel {
			fmt.Fprint(CTX.w, "{...}")
			return
		}
		fmt.Fprintln(CTX.w, "{")
		for i, k := range keys {
			v, _ := obj.Get(k)
			fmt.Fprintf(CTX.w, "%-s%-s: ", CTX.indent(level+1), k)
			CTX.printValue(v, level+1, false)
			if i < len(keys)-1 {
				fmt.Fprintf(CTX.w, ",")
			}
			fmt.Fprintln(CTX.w)
		}
		if inArray {
			level--
		}
		fmt.Fprintf(CTX.w, "%-s}", CTX.indent(level))

	case "Function":
		// Use toString() to display the argument list if possible.
		if robj, err := obj.Call("toString"); err != nil {
			fmt.Fprint(CTX.w, FunctionColor("function()"))
		} else {
			desc := strings.Trim(strings.Split(robj.String(), "{")[0], " \t\n")
			desc = strings.Replace(desc, " (", "(", 1)
			fmt.Fprint(CTX.w, FunctionColor("%-s", desc))
		}

	case "RegExp":
		fmt.Fprint(CTX.w, StringColor("%-s", toString(obj)))

	default:
		if v, _ := obj.Get("toString"); v.IsFunction() && level <= maxPrettyPrintLevel {
			s, _ := obj.Call("toString")
			fmt.Fprintf(CTX.w, "<%-s %-s>", obj.Class(), s.String())
		} else {
			fmt.Fprintf(CTX.w, "<%-s>", obj.Class())
		}
	}
}

func (CTX ppCTX) fields(obj *otto.Object) []string {
	var (
		vals, methods []string
		seen          = make(map[string]bool)
	)
	add := func(k string) {
		if seen[k] || boringKeys[k] || strings.HasPrefix(k, "_") {
			return
		}
		seen[k] = true
		if v, _ := obj.Get(k); v.IsFunction() {
			methods = append(methods, k)
		} else {
			vals = append(vals, k)
		}
	}
	iterOwnAndConstructorKeys(CTX.vm, obj, add)
	sort.Strings(vals)
	sort.Strings(methods)
	return append(vals, methods...)
}

func iterOwnAndConstructorKeys(vmPtr *otto.Otto, obj *otto.Object, f func(string)) {
	seen := make(map[string]bool)
	iterOwnKeys(vm, obj, func(prop string) {
		seen[prop] = true
		f(prop)
	})
	if cp := constructorPrototype(obj); cp != nil {
		iterOwnKeys(vm, cp, func(prop string) {
			if !seen[prop] {
				f(prop)
			}
		})
	}
}

func iterOwnKeys(vmPtr *otto.Otto, obj *otto.Object, f func(string)) {
	Object, _ := vmPtr.Object("Object")
	rv, _ := Object.Call("getOwnPropertyNames", obj.Value())
	gv, _ := rv.Export()
	switch gv := gv.(type) {
	case []interface{}:
		for _, v := range gv {
			f(v.(string))
		}
	case []string:
		for _, v := range gv {
			f(v)
		}
	default:
		panic(fmt.Errorf("Object.getOwnPropertyNames returned unexpected type %T", gv))
	}
}

func (CTX ppCTX) isBigNumber(v *otto.Object) bool {
	// Handle numbers with custom constructor.
	if v, _ := v.Get("constructor"); v.Object() != nil {
		if strings.HasPrefix(toString(v.Object()), "function BigNumber") {
			return true
		}
	}
	// Handle default constructor.
	BigNumber, _ := CTX.vmPtr.Object("BigNumber.prototype")
	if BigNumber == nil {
		return false
	}
	bv, _ := BigNumber.Call("isPrototypeOf", v)
	b, _ := bv.ToBoolean()
	return b
}

func toString(obj *otto.Object) string {
	s, _ := obj.Call("toString")
	return s.String()
}

func constructorPrototype(obj *otto.Object) *otto.Object {
	if v, _ := obj.Get("constructor"); v.Object() != nil {
		if v, _ = v.Object().Get("prototype"); v.Object() != nil {
			return v.Object()
		}
	}
	return nil
}
