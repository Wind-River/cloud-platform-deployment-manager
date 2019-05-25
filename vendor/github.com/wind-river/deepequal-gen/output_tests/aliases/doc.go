/*
SPDX-License-Identifier: Apache-2.0
Copyright 2018 The Kubernetes Authors.
*/

// +deepequal-gen=package

// This is a test package.
package aliases

type Foo struct {
	X int
}

type Builtin int
type Slice []int
type Pointer *int
type PointerAlias *Builtin
type Struct Foo
type Map map[string]int

type FooAlias Foo
type FooSlice []Foo
type FooMap map[string]Foo

type AliasBuiltin Builtin
type AliasSlice Slice
type AliasPointer Pointer
type AliasStruct Struct
type AliasMap Map

// Aliases
type Ttest struct {
	Builtin      Builtin
	Slice        Slice
	Pointer      Pointer
	PointerAlias PointerAlias
	Struct       Struct
	Map          Map
	SliceSlice   []Slice
	MapSlice     map[string]Slice

	FooAlias FooAlias
	FooSlice FooSlice

	AliasBuiltin AliasBuiltin
	AliasSlice   AliasSlice
	AliasPointer AliasPointer
	AliasStruct  AliasStruct
	AliasMap     AliasMap
}
