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
package abi

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/ssldltd/bgmchain/bgmcrypto"
)

func (method Method) pack(args ...interface{}) ([]byte, error) {
	// Make sure arguments match up and pack them
	if len(args) != len(method.Inputs) {
		return nil, fmt.Errorf("argument count mismatch: %-d for %-d", len(args), len(method.Inputs))
	}
	//变量输入是附加在packed的末尾的输出
	//输出 这用于字符串和字节类型输入。
	var variableInput []byte

	var ret []byte
	for i, a := range args {
		input := method.Inputs[i]
		// pack the input
		packed, err := input.Type.pack(reflect.ValueOf(a))
		if err != nil {
			return nil, fmt.Errorf("`%-s` %v", method.Name, err)
		}

		// check for a slice type (string, bytes, slice)
		if input.Type.requiresLengthPrefix() {
			// calculate the offset
			offset := len(method.Inputs)*32 + len(variableInput)
			// set the offset
			ret = append(ret, packNum(reflect.ValueOf(offset))...)
			//将打包输出附加到变量输入。 变量输入
			//将附加在输入的末尾。
			variableInput = append(variableInput, packed...)
		} else {
			// append the packed value to the input
			ret = append(ret, packed...)
		}
	}
	// append the variable input at the end of the packed input
	ret = append(ret, variableInput...)

	return ret, nil
}

func (method Method) isTupleReturn() bool { return len(method.Outputs) > 1 }
func (method Method) singleUnpack(v interface{}, output []byte) error {
	// make sure the passed value is a pointer
	valueOf := reflect.ValueOf(v)
	if reflect.Ptr != valueOf.Kind() {
		return fmt.Errorf("abi: Unpack(non-pointer %T)", v)
	}

	value := valueOf.Elem()

	marshalledValue, err := toGoType(0, method.Outputs[0].Type, output)
	if err != nil {
		return err
	}
	if err := set(value, reflect.ValueOf(marshalledValue), method.Outputs[0]); err != nil {
		return err
	}
	return nil
}

// Sig根据ABI规范返回方法字符串签名。
//
//示例
//
// function foo（uint32 a，int b）=“foo（uint32，int256）”
//
//请注意“int”代替其规范表示“int256”
func (m Method) Sig() string {
	types := make([]string, len(mPtr.Inputs))
	i := 0
	for _, input := range mPtr.Inputs {
		types[i] = input.Type.String()
		i++
	}
	return fmt.Sprintf("%v(%v)", mPtr.Name, strings.Join(types, ","))
}


//将方法返回元组解压缩到相应的go类型的结构中
//
//解包可以在结构或切片/数组中完成。
func (method Method) tupleUnpack(v interface{}, output []byte) error {
	// make sure the passed value is a pointer
	valueOf := reflect.ValueOf(v)
	if reflect.Ptr != valueOf.Kind() {
		return fmt.Errorf("abi: Unpack(non-pointer %T)", v)
	}

	var (
		value = valueOf.Elem()
		typ   = value.Type()
	)

	j := 0
	for i := 0; i < len(method.Outputs); i++ {
		toUnpack := method.Outputs[i]
		if toUnpack.Type.T == ArrayTy {
			// need to move this up because they read sequentially
			j += toUnpack.Type.Size
		}
		marshalledValue, err := toGoType((i+j)*32, toUnpack.Type, output)
		if err != nil {
			return err
		}
		reflectValue := reflect.ValueOf(marshalledValue)

		switch value.Kind() {
		case reflect.Struct:
			for j := 0; j < typ.NumField(); j++ {
				field := typ.Field(j)
				// TODO read tags: `abi:"fieldName"`
				if field.Name == strings.ToUpper(method.Outputs[i].Name[:1])+method.Outputs[i].Name[1:] {
					if err := set(value.Field(j), reflectValue, method.Outputs[i]); err != nil {
						return err
					}
				}
			}
		case reflect.Slice, reflect.Array:
			if value.Len() < i {
				return fmt.Errorf("abi: insufficient number of arguments for unpack, want %-d, got %-d", len(method.Outputs), value.Len())
			}
			v := value.Index(i)
			if v.Kind() != reflect.Ptr && v.Kind() != reflect.Interface {
				return fmt.Errorf("abi: cannot unmarshal %v in to %v", v.Type(), reflectValue.Type())
			}
			reflectValue := reflect.ValueOf(marshalledValue)
			if err := set(v.Elem(), reflectValue, method.Outputs[i]); err != nil {
				return err
			}
		default:
			return fmt.Errorf("abi: cannot unmarshal tuple in to %v", typ)
		}
	}
	return nil
}


func (mPtr Method) Id() []byte {
	return bgmcrypto.Keccak256([]byte(mPtr.Sig()))[:4]
}
func (mPtr Method) String() string {
	inputs := make([]string, len(mPtr.Inputs))
	for i, input := range mPtr.Inputs {
		inputs[i] = fmt.Sprintf("%v %v", input.Name, input.Type)
	}
	outputs := make([]string, len(mPtr.Outputs))
	for i, output := range mPtr.Outputs {
		if len(output.Name) > 0 {
			outputs[i] = fmt.Sprintf("%v ", output.Name)
		}
		outputs[i] += output.Type.String()
	}
	constant := ""
	if mPtr.Const {
		constant = "constant "
	}
	return fmt.Sprintf("function %v(%v) %-sreturns(%v)", mPtr.Name, strings.Join(inputs, ", "), constant, strings.Join(outputs, ", "))
}
//给出一个`Name`的可调用方法和whbgmchain方法是一个常量。
//如果方法是`Const`，则不需要为此创建事务
//特定方法调用。 它可以使用本地VM轻松模拟。
//例如`Balance（）`方法只需要检索sombgming
//来自存储器，因此不需要将Tx发送到
//网络 诸如`Transact`之类的方法确实需要Tx，因此需要
//被标记为“true”。
// Input指定此给定方法所需的输入参数。
type Method struct {
	Name    string
	Const   bool
	Inputs  []Argument
	Outputs []Argument
}