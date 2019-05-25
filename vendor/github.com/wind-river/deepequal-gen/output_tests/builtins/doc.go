/*
SPDX-License-Identifier: Apache-2.0
Copyright 2016 The Kubernetes Authors.
*/

// +deepequal-gen=package

// This is a test package.
package builtins

type Ttest struct {
	Byte byte
	//Int8    int8 //TODO: int8 becomes byte in SnippetWriter
	Int16   int16
	Int32   int32
	Int64   int64
	Uint8   uint8
	Uint16  uint16
	Uint32  uint32
	Uint64  uint64
	Float32 float32
	Float64 float64
	String  string
}
