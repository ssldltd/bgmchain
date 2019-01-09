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

/*
Package vm implements the Bgmchain Virtual Machine.

The vm package implements two EVMs, a byte code VM and a JIT VmPtr. The BC
(Byte Code) VM loops over a set of bytes and executes them according to the set
of rules defined in the Bgmchain yellow paper. When the BC VM is invoked it
invokes the JIT VM in a separate goroutine and compiles the byte code in JIT
instructions.

*/
package vm
