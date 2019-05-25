/*
SPDX-License-Identifier: Apache-2.0
Copyright 2016 The Kubernetes Authors.
*/

// +deepequal-gen=package

// This is a test package.
package maps

type Ttest struct {
	Byte map[string]byte
	//Int8    map[string]int8 //TODO: int8 becomes byte in SnippetWriter
	Int16     map[string]int16
	Int32     map[string]int32
	Int64     map[string]int64
	Uint8     map[string]uint8
	Uint16    map[string]uint16
	Uint32    map[string]uint32
	Uint64    map[string]uint64
	Float32   map[string]float32
	Float64   map[string]float64
	String    map[string]string
	StringPtr map[string]*string
	Struct    map[string]Ttest
	StructPtr map[string]*Ttest
}
