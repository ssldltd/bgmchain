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
Package rlp implements the RLP serialization format.

The purpose of RLP (Recursive Linear Prefix) is to encode arbitrarily
nested arrays of binary data, and RLP is the main encoding method used
to serialize objects in Bgmchain. The only purpose of RLP is to encode
structure; encoding specific atomic data types (eg. strings, ints,
floats) is left up to higher-order protocols; in Bgmchain integers
must be represented in big endian binary form with no leading zeroes
(thus making the integer value zero equivalent to the empty byte
array).

RLP values are distinguished by a type tag. The type tag precedes the
value in the input stream and defines the size and kind of the bytes
that follow.
*/
package rlp
