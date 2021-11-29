// Copyright 2020 Azul Systems, Inc. All rights reserved.
// Use of this source code is governed by the 3-Clause BSD
// license that can be found in the LICENSE file.

package main

import "encoding/binary"

var bigEndian = binary.BigEndian

type ClassFileReader struct {
	bytes []byte
}

func CreateClassFileReader(bytes []byte) *ClassFileReader {
	return &ClassFileReader{bytes: bytes}
}

func (r *ClassFileReader) ReadUint32() uint32 {
	value := bigEndian.Uint32(r.bytes[:4])
	r.bytes = r.bytes[4:]
	return value
}

func (r *ClassFileReader) ReadUint16() uint16 {
	value := bigEndian.Uint16(r.bytes[:2])
	r.bytes = r.bytes[2:]
	return value
}

func (r *ClassFileReader) ReadUint8() uint8 {
	return r.ReadBytes(1)[0]
}

func (r *ClassFileReader) ReadBytes(len int) []byte {
	bytes := r.bytes[:len]
	r.bytes = r.bytes[len:]
	return bytes
}

func (r *ClassFileReader) Size() int {
	return len(r.bytes)
}
