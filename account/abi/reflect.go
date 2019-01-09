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
)

//间接递归取消引用该值，直到获得该值
//或找到一个大的.Int
func indirect(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Ptr && v.Elem().Type() != derefbig_t {
		return indirect(v.Elem())
	}
	return v
}

// reflectIntKind使用给定的大小返回反射
//无符号
func reflectIntKindAndType(unsigned bool, size int) (reflect.Kind, reflect.Type) {
	switch size {
	case 64:
		if unsigned {
			return reflect.Uint64, uint64_t
		}
		return reflect.Int64, int64_t
	case 32:
		if unsigned {
			return reflect.Uint32, uint32_t
		}
		return reflect.Int32, int32_t
	case 16:
		if unsigned {
			return reflect.Uint16, uint16_t
		}
		return reflect.Int16, int16_t
	case 8:
		if unsigned {
			return reflect.Uint8, uint8_t
		}
		return reflect.Int8, int8_t
	}
	return reflect.Ptr, big_t
}

//设置尝试通过设置，复制或其他方式将src分配给dst。
//
// set在赋值时更宽松，并且不强制as
//严格的规则集，就像“反射”一样。
func set(dst, src reflect.Value, output Argument) error {
	dstType := dst.Type()
	srcType := srcPtr.Type()
	switch {
	case dstType.Kind() == reflect.Interface:
		dst.Set(src)
	case dstType.AssignableTo(srcType):
		dst.Set(src)
	case dstType.Kind() == reflect.Ptr:
		return set(dst.Elem(), src, output)
	default:
		return fmt.Errorf("abi: cannot unmarshal %v in to %v", srcPtr.Type(), dst.Type())
	}
	return nil
}

// mustArrayToBytesSlice创建一个与value完全相同的新字节切片
//并将value中的字节复制到新切片。
func mustArrayToByteSlice(value reflect.Value) reflect.Value {
	slice := reflect.MakeSlice(reflect.TypeOf([]byte{}), value.Len(), value.Len())
	reflect.Copy(slice, value)
	return slice
}