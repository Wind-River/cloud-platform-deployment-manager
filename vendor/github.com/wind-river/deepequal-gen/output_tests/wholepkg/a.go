/*
SPDX-License-Identifier: Apache-2.0
Copyright 2016 The Kubernetes Authors.
*/

package wholepkg

import (
	"reflect"
)

// Trivial
type StructEmpty struct{}

// Only primitives
type StructPrimitives struct {
	BoolField   bool
	IntField    int
	StringField string
	FloatField  float64
}
type StructPrimitivesAlias StructPrimitives
type StructEmbedStructPrimitives struct {
	StructPrimitives
}
type StructEmbedInt struct {
	int
}
type StructStructPrimitives struct {
	StructField StructPrimitives
}

// Manual DeepEqual method
type ManualStruct struct {
	StringField string
}

func (in *ManualStruct) DeepEqual(other *ManualStruct) bool {
	return other != nil && in.StringField == other.StringField
}

type ManualStructAlias ManualStruct

type StructEmbedManualStruct struct {
	ManualStruct
}

// Only pointers to primitives
type StructPrimitivePointers struct {
	BoolPtrField   *bool
	IntPtrField    *int
	StringPtrField *string
	FloatPtrField  *float64
}
type StructPrimitivePointersAlias StructPrimitivePointers
type StructEmbedStructPrimitivePointers struct {
	StructPrimitivePointers
}
type StructEmbedPointer struct {
	*int
}
type StructStructPrimitivePointers struct {
	StructField StructPrimitivePointers
}

// Manual DeepCopy method
type ManualSlice []string

func (in *ManualSlice) DeepEqual(other *ManualSlice) bool {
	return reflect.DeepEqual(in, other)
}

// Slices
type StructSlices struct {
	SliceBoolField                         []bool
	SliceByteField                         []byte
	SliceIntField                          []int
	SliceStringField                       []string
	SliceFloatField                        []float64
	SliceStructPrimitivesField             []StructPrimitives
	SliceStructPrimitivesAliasField        []StructPrimitivesAlias
	SliceStructPrimitivePointersField      []StructPrimitivePointers
	SliceStructPrimitivePointersAliasField []StructPrimitivePointersAlias
	SliceManualStructField                 []ManualStruct
	ManualSliceField                       ManualSlice
}
type StructSlicesAlias StructSlices
type StructEmbedStructSlices struct {
	StructSlices
}
type StructStructSlices struct {
	StructField StructSlices
}

// Everything
type StructEverything struct {
	BoolField                 bool
	IntField                  int
	StringField               string
	FloatField                float64
	StructField               StructPrimitives
	EmptyStructField          StructEmpty
	ManualStructField         ManualStruct
	ManualStructAliasField    ManualStructAlias
	BoolPtrField              *bool
	IntPtrField               *int
	StringPtrField            *string
	FloatPtrField             *float64
	PrimitivePointersField    StructPrimitivePointers
	ManualStructPtrField      *ManualStruct
	ManualStructAliasPtrField *ManualStructAlias
	SliceBoolField            []bool
	SliceByteField            []byte
	SliceIntField             []int
	SliceStringField          []string
	SliceFloatField           []float64
	SlicesField               StructSlices
	SliceManualStructField    []ManualStruct
	ManualSliceField          ManualSlice
}

// An Object
type StructExplicitObject struct {
	x int
}

// An Object which is used a non-pointer
type StructNonPointerExplicitObject struct {
	x int
}

// +deepequal-gen=false
type StructTypeMeta struct {
}

type StructObjectAndList struct {
}

type StructObjectAndObject struct {
}

type StructExplicitSelectorExplicitObject struct {
	StructTypeMeta
}
