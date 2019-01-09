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
	"encoding/binary"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ssldltd/bgmchain/bgmcommon"
)


// reads the integer based on its kind
func readInteger(kind reflect.Kind, b []byte) interface{} {
	switch kind {
	case reflect.Uint32:
		return binary.BigEndian.Uint32(b[len(b)-4:])
	case reflect.Uint8:
		return b[len(b)-1]
	case reflect.Uint16:
		return binary.BigEndian.Uint16(b[len(b)-2:])
	case reflect.Uint64:
		return binary.BigEndian.Uint64(b[len(b)-8:])
	case reflect.Int8:
		return int8(b[len(b)-1])
	case reflect.Int16:
		return int16(binary.BigEndian.Uint16(b[len(b)-2:]))
	case reflect.Int32:
		return int32(binary.BigEndian.Uint32(b[len(b)-4:]))
	case reflect.Int64:
		return int64(binary.BigEndian.Uint64(b[len(b)-8:]))
	default:
		return new(big.Int).SetBytes(b)
	}
}

// reads a bool
func readBool(word []byte) (bool, error) {
	for _, b := range word[:31] {
		if b != 0 {
			return false, errBadBool
		}
	}
	switch word[31] {
	case 1:
		return true, nil
	case 0:
		return false, nil
	default:
		return false, errBadBool
	}
}

//函数类型只是末尾带有函数选择签名的地址。
//通过始终将其标记为24个数组（地址+ sig = 24个字节）来强制执行该标准
func readFunctionType(t Type, word []byte) (funcTy [24]byte, err error) {
	if tPtr.T != FunctionTy {
		return [24]byte{}, fmt.Errorf("abi: invalid type in call to make function type byte array.")
	}
	if garbage := binary.BigEndian.Uint64(word[24:32]); garbage != 0 {
		err = fmt.Errorf("abi: got improperly encoded function type, got %v", word)
	} else {
		copy(funcTy[:], word[0:24])
	}
	return
}

//通过反射，创建一个要读取的固定数组
func readFixedBytes(t Type, word []byte) (interface{}, error) {
	if tPtr.T != FixedBytesTy {
		return nil, fmt.Errorf("abi: invalid type in call to make fixed byte array.")
	}
	// convert
	array := reflect.New(tPtr.Type).Elem()

	reflect.Copy(array, reflect.ValueOf(word[0:tPtr.Size]))
	return array.Interface(), nil

}

// iteratively unpack elements
func forEachUnpack(t Type, output []byte, start, size int) (interface{}, error) {
	if start+32*size > len(output) {
		return nil, fmt.Errorf("abi: cannot marshal in to go array: offset %-d would go over slice boundary (len=%-d)", len(output), start+32*size)
	}

	//这个值将成为我们的切片或我们的数组，具体取决于类型
	var refSlice reflect.Value
	slice := output[start : start+size*32]

	if tPtr.T == SliceTy {
		// declare our slice
		refSlice = reflect.MakeSlice(tPtr.Type, size, size)
	} else if tPtr.T == ArrayTy {
		// declare our array
		refSlice = reflect.New(tPtr.Type).Elem()
	} else {
		return nil, fmt.Errorf("abi: invalid type in array/slice unpacking stage")
	}

	for i, j := start, 0; j*32 < len(slice); i, j = i+32, j+1 {
		// this corrects the arrangement so that we get all the underlying array values
		if tPtr.ElemPtr.T == ArrayTy && j != 0 {
			i = start + tPtr.ElemPtr.Size*32*j
		}
		inter, err := toGoType(i, *tPtr.Elem, output)
		if err != nil {
			return nil, err
		}
		// append the item to our reflect slice
		refSlice.Index(j).Set(reflect.ValueOf(inter))
	}

	// return the interface
	return refSlice.Interface(), nil
}

// toGoType解析输出字节并递归地分配这些字节的值
//进入符合ABI规范的go类型。
func toGoType(index int, t Type, output []byte) (interface{}, error) {
	if index+32 > len(output) {
		return nil, fmt.Errorf("abi: cannot marshal in to go type: length insufficient %-d require %-d", len(output), index+32)
	}

	var (
		returnOutput []byte
		begin, end   int
		err          error
	)

	//如果我们需要长度前缀，请找到返回的开头字和大小。
	if tPtr.requiresLengthPrefix() {
		begin, end, err = lengthPrefixPointsTo(index, output)
		if err != nil {
			return nil, err
		}
	} else {
		returnOutput = output[index : index+32]
	}

	switch tPtr.T {
	case FunctionTy:
		return readFunctionType(t, returnOutput)
	case SliceTy:
		return forEachUnpack(t, output, begin, end)
	case ArrayTy:
		return forEachUnpack(t, output, index, tPtr.Size)
	case StringTy: // variable arrays are written at the end of the return bytes
		return string(output[begin : begin+end]), nil
	case FixedBytesTy:
		return readFixedBytes(t, returnOutput)
	case IntTy, UintTy:
		return readInteger(tPtr.Kind, returnOutput), nil
	case BoolTy:
		return readBool(returnOutput)
	case AddressTy:
		return bgmcommon.BytesToAddress(returnOutput), nil
	case HashTy:
		return bgmcommon.BytesToHash(returnOutput), nil
	case BytesTy:
		return output[begin : begin+end], nil
	default:
		return nil, fmt.Errorf("abi: unknown type %v", tPtr.T)
	}
}

//将32字节切片解释为偏移量，然后确定要对哪种类型的索引进行解码。
func lengthPrefixPointsTo(index int, output []byte) (start int, length int, err error) {
	offset := int(binary.BigEndian.Uint64(output[index+24 : index+32]))
	if offset+32 > len(output) {
		return 0, 0, fmt.Errorf("abi: cannot marshal in to go slice: offset %-d would go over slice boundary (len=%-d)", len(output), offset+32)
	}
	length = int(binary.BigEndian.Uint64(output[offset+24 : offset+32]))
	if offset+32+length > len(output) {
		return 0, 0, fmt.Errorf("abi: cannot marshal in to go type: length insufficient %-d require %-d", len(output), offset+32+length)
	}
	start = offset + 32

	//fmt.Printf("LENGTH PREFIX INFO: \nsize: %v\noffset: %v\nstart: %v\n", length, offset, start)
	return
}

// checks for proper formatting of byte output
func bytesAreProper(output []byte) error {
	if len(output) == 0 {
		return fmt.Errorf("abi: unmarshalling empty output")
	} else if len(output)%32 != 0 {
		return fmt.Errorf("abi: improperly formatted output")
	} else {
		return nil
	}
}
// unpacker是一个实用程序界面，使我们能够拥有
//事件和方法之间的抽象以及正确的抽象
//“解包”他们; 例如 事件使用输入，方法使用输出。
type unpacker interface {
	tupleUnpack(v interface{}, output []byte) error
	singleUnpack(v interface{}, output []byte) error
	isTupleReturn() bool
}