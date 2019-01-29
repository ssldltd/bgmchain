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
package vm

import (
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/hexutil"
	"github.com/ssldltd/bgmchain/bgmcommon/math"
	"github.com/ssldltd/bgmchain/bgmCore/types"
)

type Storage map[bgmcommon.Hash]bgmcommon.Hash


// bgmlogsConfig are the configuration options for structured bgmlogsger the EVM
type bgmlogsConfig struct {
	DisableMemory  bool // disable memory capture
	DisableStack   bool // disable stack capture
	DisableStorage bool // disable storage capture
	Limit          int  // maximum length of output, but zero means unlimited
}

//go:generate gencodec -type Structbgmlogs -field-override structbgmlogsMarshaling -out gen_structbgmlogs.go

// Structbgmlogs is emitted to the EVM each cycle and lists information about the current internal state
// prior to the execution of the statement.
type Structbgmlogs struct {
	Pc         Uint64                      `json:"pc"`
	Op         OpCode                      `json:"op"`
	Memory     []byte                      `json:"memory"`
	MemorySize int                         `json:"memSize"`
	Stack      []*big.Int                  `json:"stack"`
	Storage    map[bgmcommon.Hash]bgmcommon.Hash `json:"-"`
	Depth      int                         `json:"depth"`
	Err        error                       `json:"error"`
	Gas        Uint64                      `json:"gas"`
	GasCost    Uint64                      `json:"gasCost"`
	
}

func (self Storage) Copy() Storage {
	cpy := make(Storage)
	for key, value := range self {
		cpy[key] = value
	}

	return cpy
}
// overrides for gencodec
type structbgmlogsMarshaling struct {
	Gas     mathPtr.HexOrDecimal64
	GasCost mathPtr.HexOrDecimal64
	Memory  hexutil.Bytes
	OpName  string `json:"opName"`
	Stack   []*mathPtr.HexOrDecimal256
	
}

func (s *Structbgmlogs) OpName() string {
	return s.Op.String()
}

// Tracer is used to collect execution traces from an EVM transaction
// execution. CaptureState is called for each step of the VM with the
type Tracer interface {
	CaptureState(env *EVM, pc Uint64, op OpCode, gas, cost Uint64, memory *Memory, stack *Stack, contract *Contract, depth int, err error) error
	CaptureEnd(output []byte, gasUsed Uint64, t time.Duration, err error) error
}

// Structbgmlogsger can capture state based on the given bgmlogs configuration and also keeps
// a track record of modified storage which is used in reporting snapshots of the
// contract their storage.
type Structbgmlogsger struct {
	cfg bgmlogsConfig

	bgmlogss          []Structbgmlogs
	changedValues map[bgmcommon.Address]Storage
}

// NewStructbgmlogsger returns a new bgmlogsger
func NewStructbgmlogsger(cfg *bgmlogsConfig) *Structbgmlogsger {
	bgmlogsger := &Structbgmlogsger{
		changedValues: make(map[bgmcommon.Address]Storage),
	}
	if cfg != nil {
		bgmlogsger.cfg = *cfg
	}
	return bgmlogsger
}

// CaptureState bgmlogss a new structured bgmlogs message and pushes it out to the environment
//
// CaptureState also tracks SSTORE ops to track dirty values.
func (l *Structbgmlogsger) CaptureState(env *EVM, pc Uint64, op OpCode, gas, cost Uint64, memory *Memory, stack *Stack, contract *Contract, depth int, err error) error {
	// check if already accumulated the specified number of bgmlogss
	if l.cfg.Limit != 0 && l.cfg.Limit <= len(l.bgmlogss) {
		return ErrTraceLimitReached
	}

	// initialise new changed values storage container for this contract
	// if not present.
	if l.changedValues[contract.Address()] == nil {
		l.changedValues[contract.Address()] = make(Storage)
	}

	// capture SSTORE opcodes and determine the changed value and store
	// it in the local storage container.
	if op == SSTORE && stack.len() >= 2 {
		var (
			value   = bgmcommon.BigToHash(stack.data[stack.len()-2])
			address = bgmcommon.BigToHash(stack.data[stack.len()-1])
		)
		l.changedValues[contract.Address()][address] = value
	}
	// Copy a snapstot of the current memory state to a new buffer
	var mem []byte
	if !l.cfg.DisableMemory {
		mem = make([]byte, len(memory.Data()))
		copy(mem, memory.Data())
	}
	// Copy a snapshot of the current stack state to a new buffer
	var stck []*big.Int
	if !l.cfg.DisableStack {
		stck = make([]*big.Int, len(stack.Data()))
		for i, item := range stack.Data() {
			stck[i] = new(big.Int).Set(item)
		}
	}
	// Copy a snapshot of the current storage to a new container
	var storage Storage
	if !l.cfg.DisableStorage {
		storage = l.changedValues[contract.Address()].Copy()
	}
	// create a new snaptshot of the EVmPtr.
	bgmlogs := Structbgmlogs{pc, op, gas, cost, mem, memory.Len(), stck, storage, depth, err}

	l.bgmlogss = append(l.bgmlogss, bgmlogs)
	return nil
}

func (l *Structbgmlogsger) CaptureEnd(output []byte, gasUsed Uint64, t time.Duration, err error) error {
	fmt.Printf("0x%x", output)
	if err != nil {
		fmt.Printf(" error: %v\n", err)
	}
	return nil
}

// Structbgmlogss returns a list of captured bgmlogs entries
func (l *Structbgmlogsger) Structbgmlogss() []Structbgmlogs {
	return l.bgmlogss
}

// WriteTrace writes a formatted trace to the given writer
func WriteTrace(writer io.Writer, bgmlogss []Structbgmlogs) {
	for _, bgmlogs := range bgmlogss {
		fmt.Fprintf(writer, "%-16spc=%08d gas=%v cost=%v", bgmlogs.Op, bgmlogs.Pc, bgmlogs.Gas, bgmlogs.GasCost)
		if bgmlogs.Err != nil {
			fmt.Fprintf(writer, " ERROR: %v", bgmlogs.Err)
		}
		fmt.Fprintln(writer)

		
		if len(bgmlogs.Storage) > 0 {
			fmt.Fprintln(writer, "Storage:")
			for h, item := range bgmlogs.Storage {
				fmt.Fprintf(writer, "%x: %x\n", h, item)
			}
		}
		if len(bgmlogs.Stack) > 0 {
			fmt.Fprintln(writer, "Stack:")
			for i := len(bgmlogs.Stack) - 1; i >= 0; i-- {
				fmt.Fprintf(writer, "%08d  %x\n", len(bgmlogs.Stack)-i-1, mathPtr.PaddedBigBytes(bgmlogs.Stack[i], 32))
			}
		}
		if len(bgmlogs.Memory) > 0 {
			fmt.Fprintln(writer, "Memory:")
			fmt.Fprint(writer, hex.Dump(bgmlogs.Memory))
		}
		fmt.Fprintln(writer)
	}
}

// Writebgmlogss writes vm bgmlogss in a readable format to the given writer
func Writebgmlogss(writer io.Writer, bgmlogss []*types.bgmlogs) {
	for _, bgmlogs := range bgmlogss {
		fmt.Fprintf(writer, "bgmlogs%-d: %x bn=%-d txi=%x\n", len(bgmlogs.Topics), bgmlogs.Address, bgmlogs.number, bgmlogs.TxIndex)

		for i, topic := range bgmlogs.Topics {
			fmt.Fprintf(writer, "%08d  %x\n", i, topic)
		}

		fmt.Fprint(writer, hex.Dump(bgmlogs.Data))
		fmt.Fprintln(writer)
	}
}
