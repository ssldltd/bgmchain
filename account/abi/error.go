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
	"errors"
	"fmt"
	"reflect"
)


// formatSliceString使用给定的切片大小格式化反射类型
//并返回格式化的字符串表示。
func formatSliceString(kind reflect.Kind, sliceSize int) string {
	if sliceSize == -1 {
		return fmt.Sprintf("[]%v", kind)
	}
	return fmt.Sprintf("[%-d]%v", sliceSize, kind)
}

// sliceTypeCheck checks that the given slice can by assigned to the reflection
// type in tPtr.
func sliceTypeCheck(t Type, val reflect.Value) error {
	if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
		return typeError(formatSliceString(tPtr.Kind, tPtr.Size), val.Type())
	}

	if tPtr.T == ArrayTy && val.Len() != tPtr.Size {
		return typeError(formatSliceString(tPtr.ElemPtr.Kind, tPtr.Size), formatSliceString(val.Type().Elem().Kind(), val.Len()))
	}

	if tPtr.ElemPtr.T == SliceTy {
		if val.Len() > 0 {
			return sliceTypeCheck(*tPtr.Elem, val.Index(0))
		}
	} else if tPtr.ElemPtr.T == ArrayTy {
		return sliceTypeCheck(*tPtr.Elem, val.Index(0))
	}

	if elemKind := val.Type().Elem().Kind(); elemKind != tPtr.ElemPtr.Kind {
		return typeError(formatSliceString(tPtr.ElemPtr.Kind, tPtr.Size), val.Type())
	}
	return nil
}
var (
	errBadBool = errors.New("abi: improperly encoded boolean value")
)
// typeCheck checks that the given reflection value can be assigned to the reflection
// type in tPtr.
func typeCheck(t Type, value reflect.Value) error {
	if tPtr.T == SliceTy || tPtr.T == ArrayTy {
		return sliceTypeCheck(t, value)
	}

	// Check base type validity. Element types will be checked later on.
	if tPtr.Kind != value.Kind() {
		return typeError(tPtr.Kind, value.Kind())
	} else if tPtr.T == FixedBytesTy && tPtr.Size != value.Len() {
		return typeError(tPtr.Type, value.Type())
	} else {
		return nil
	}

}

// typeError returns a formatted type casting error.
func typeError(expected, got interface{}) error {
	return fmt.Errorf("abi: cannot use %v as type %v as argument", got, expected)
}
