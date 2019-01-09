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

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcrypto"
)

//事件是可能由EVM的bgmlogs机制触发的事件。 事件
//保存有关输出输出的类型信息（输入）。 匿名事件
//不要将签名规范表示作为第一个bgmlogs主题。
type Event struct {
	Name      string
	Anonymous bool
	Inputs    []Argument
}

// Id返回由...使用的事件签名的规范表示
// abi定义以标识事件名称和类型。
func (e Event) Id() bgmcommon.Hash {
	types := make([]string, len(e.Inputs))
	i := 0
	for _, input := range e.Inputs {
		types[i] = input.Type.String()
		i++
	}
	return bgmcommon.BytesToHash(bgmcrypto.Keccak256([]byte(fmt.Sprintf("%v(%v)", e.Name, strings.Join(types, ",")))))
}

//将事件返回元组解压缩到相应的go类型的结构中
//
//解包可以在结构或切片/数组中完成。
func (e Event) tupleUnpack(v interface{}, output []byte) error {
	// make sure the passed value is a pointer
	valueOf := reflect.ValueOf(v)
	if reflect.Ptr != valueOf.Kind() {
		return fmt.Errorf("abi: Unpack(non-pointer %T)", v)
	}

	var (
		value = valueOf.Elem()
		typ   = value.Type()
	)

	if value.Kind() != reflect.Struct {
		return fmt.Errorf("abi: cannot unmarshal tuple in to %v", typ)
	}

	j := 0
	for i := 0; i < len(e.Inputs); i++ {
		input := e.Inputs[i]
		if input.Indexed {
			// can't read, continue
			continue
		} else if input.Type.T == ArrayTy {
			// need to move this up because they read sequentially
			j += input.Type.Size
		}
		marshalledValue, err := toGoType((i+j)*32, input.Type, output)
		if err != nil {
			return err
		}
		reflectValue := reflect.ValueOf(marshalledValue)

		switch value.Kind() {
		case reflect.Struct:
			for j := 0; j < typ.NumField(); j++ {
				field := typ.Field(j)
				// TODO read tags: `abi:"fieldName"`
				if field.Name == strings.ToUpper(e.Inputs[i].Name[:1])+e.Inputs[i].Name[1:] {
					if err := set(value.Field(j), reflectValue, e.Inputs[i]); err != nil {
						return err
					}
				}
			}
		case reflect.Slice, reflect.Array:
			if value.Len() < i {
				return fmt.Errorf("abi: insufficient number of arguments for unpack, want %-d, got %-d", len(e.Inputs), value.Len())
			}
			v := value.Index(i)
			if v.Kind() != reflect.Ptr && v.Kind() != reflect.Interface {
				return fmt.Errorf("abi: cannot unmarshal %v in to %v", v.Type(), reflectValue.Type())
			}
			reflectValue := reflect.ValueOf(marshalledValue)
			if err := set(v.Elem(), reflectValue, e.Inputs[i]); err != nil {
				return err
			}
		default:
			return fmt.Errorf("abi: cannot unmarshal tuple in to %v", typ)
		}
	}
	return nil
}

func (e Event) singleUnpack(v interface{}, output []byte) error {
	// make sure the passed value is a pointer
	valueOf := reflect.ValueOf(v)
	if reflect.Ptr != valueOf.Kind() {
		return fmt.Errorf("abi: Unpack(non-pointer %T)", v)
	}

	if e.Inputs[0].Indexed {
		return fmt.Errorf("abi: attempting to unpack indexed variable into element.")
	}

	value := valueOf.Elem()

	marshalledValue, err := toGoType(0, e.Inputs[0].Type, output)
	if err != nil {
		return err
	}
	if err := set(value, reflect.ValueOf(marshalledValue), e.Inputs[0]); err != nil {
		return err
	}
	return nil
}
func (e Event) isTupleReturn() bool { return len(e.Inputs) > 1 }