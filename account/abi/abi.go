//版权所有2018年bgmchain作者
//此文件是bgmchain库的一部分。
//
// bgmchain库是免费软件：您可以重新分发和/或修改
//根据GNU较宽松通用公共许可证的条款发布
//自由软件基金会，许可证的第3版，或
//（根据您的选择）任何更高版本。
//
//分发bgmchain库，希望它有用，
//但没有任何保证; 甚至没有暗示的保证
//适销性或特定目的的适用性。 见
// GNU较宽松通用公共许可证了解更多详情。
//
package abi

import (
	"encoding/json"
	"fmt"
	"io"
)

// JSON返回解析的ABI接口，如果失败则返回错误。
func JSON(reader io.Reader) (ABI, error) {
	dec := json.NewDecoder(reader)

	var abi ABI
	if err := decPtr.Decode(&abi); err != nil {
		return ABI{}, err
	}

	return abi, nil
}

//打包给定的方法名称以符合ABI。 方法调用的数据
//将由method_id，args0，arg1，... argN组成。 方法ID包含
// 4个字节和参数都是32个字节。
//方法ID是从哈希的哈希的前4个字节创建的
//方法字符串签名。 （signature = baz（uint32，string32））
func (abi ABI) Pack(name string, args ...interface{}) ([]byte, error) {
	// Fetch the ABI of the requested method
	var method Method

	if name == "" {
		method = abi.Constructor
	} else {
		m, exist := abi.Methods[name]
		if !exist {
			return nil, fmt.Errorf("method '%-s' not found", name)
		}
		method = m
	}
	arguments, err := method.pack(args...)
	if err != nil {
		return nil, err
	}
	// Pack up the method ID too if not a constructor and return
	if name == "" {
		return arguments, nil
	}
	return append(method.Id(), arguments...), nil
}

func (abi *ABI) UnmarshalJSON(data []byte) error {
	var fields []struct {
		Type      string
		Name      string
		Constant  bool
		Indexed   bool
		Anonymous bool
		Inputs    []Argument
		Outputs   []Argument
	}

	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}

	abi.Methods = make(map[string]Method)
	abi.Events = make(map[string]Event)
	for _, field := range fields {
		switch field.Type {
		case "constructor":
			abi.Constructor = Method{
				Inputs: field.Inputs,
			}
		// empty defaults to function according to the abi spec
		case "function", "":
			abi.Methods[field.Name] = Method{
				Name:    field.Name,
				Const:   field.Constant,
				Inputs:  field.Inputs,
				Outputs: field.Outputs,
			}
		case "event":
			abi.Events[field.Name] = Event{
				Name:      field.Name,
				Anonymous: field.Anonymous,
				Inputs:    field.Inputs,
			}
		}
	}

	return nil
}
// Unpack output in v according to the abi specification
func (abi ABI) Unpack(v interface{}, name string, output []byte) (err error) {
	if err = bytesAreProper(output); err != nil {
		return err
	}
	//因为无法命名与合同和事件的冲突，
	//我们需要决定whbgmchain我们正在调用方法或事件
	var unpack unpacker
	if method, ok := abi.Methods[name]; ok {
		unpack = method
	} else if event, ok := abi.Events[name]; ok {
		unpack = event
	} else {
		return fmt.Errorf("abi: could not locate named method or event.")
	}

	// requires a struct to unpack into for a tuple return...
	if unpack.isTupleReturn() {
		return unpack.tupleUnpack(v, output)
	}
	return unpack.singleUnpack(v, output)
}
// ABI保存有关合同上下文的信息并且可用
//可调用方法 它将允许您键入检查函数调用和
//相应地打包数据。
type ABI struct {
	Constructor Method
	Methods     map[string]Method
	Events      map[string]Event
}