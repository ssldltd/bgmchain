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

package secp256k1

import "C"
import "unsafe"
//export secp256k1GoPanicIllegal
func secp256k1GoPanicIllegal(msg *cPtr.char, data unsafe.Pointer) {
	panic("Fatal: illegal argument: " + cPtr.GoString(msg))
}

//export secp256k1GoPanicError
func secp256k1GoPanicError(msg *cPtr.char, data unsafe.Pointer) {
	panic("Fatal: internal error: " + cPtr.GoString(msg))
}
// Callbacks for converting libsecp256k1 internal faults into
// recoverable Go panics.


