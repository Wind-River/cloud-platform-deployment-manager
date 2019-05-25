/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package common

import (
	"github.com/imdario/mergo"
	"reflect"
)

type MergeTransformer struct {
	OverwriteSlices bool
}

// IsNumericType determines whether the type specified is one of the built-in
// numeric type values.
// from github.com/imdario/mergo
func IsNumericType(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return true
	case reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

// Transformer implements a struct merge strategy for arrays and slices.  The
// default mergo approach to merging slices is to leave them intact unless the
// AppendSlices modifier is used.  That would cause both the parent and subclass
// arrays to be concatenated together.  This transformer provides a way to
// replace individual array elements if they are found to match an element in
// the destination array.  This is only possible if the array element structs
// implement the IsKeyEqual method.
// TODO(alegacy): can these lambda functions be refactored out to standalone
//  functions.
func (t MergeTransformer) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	if IsNumericType(typ.Kind()) || typ.Kind() == reflect.Bool {
		// mergo doesn't differentiate between numeric values and pointers
		// when it comes to deciding whether to accept a zero value from the src
		// struct.  For example, if a src struct field has a numeric field value
		// of 0 then it will not overwrite the dst field because it considers
		// 0 to be unset.  In our structs if a field is optional then we
		// declare it as a pointer.  We only want the default behaviour for
		// pointers. For numeric and boolean values we want to overwrite the
		// destination because we consider those mandatory if we didn't specify
		// them as a pointer.
		return func(dst, src reflect.Value) error {
			if dst.CanSet() {
				dst.Set(src)
			}
			return nil
		}
	} else if typ.Kind() == reflect.Ptr && typ.Elem().Kind() == reflect.Struct {
		// mergo doesn't handle struct pointers how we need them to be handled.
		// Rather than simply overwrite the pointer we need the structs to be
		// merged recursively so handle it with a custom transformer.
		return func(dst, src reflect.Value) error {
			if dst.IsNil() {
				dst.Set(src)
				return nil
			} else if src.IsNil() {
				// Do nothing
				return nil
			}
			dst = dst.Elem()
			src = src.Elem()
			merge := reflect.ValueOf(mergo.Merge)
			result := merge.Call([]reflect.Value{dst.Addr(),
				src,
				reflect.ValueOf(mergo.WithOverride),
				reflect.ValueOf(mergo.WithTransformers(t))})
			if result[0].IsValid() && result[0].IsNil() == false {
				return result[0].Interface().(error)
			}
			return nil
		}
	} else if typ.Kind() == reflect.Slice {
		// mergo doesn't handle slices how we need them to be handled.  Rather
		// than simply overwrite the slice or append to the slice we need each
		// element of the slice to
		// be handled separately.  If the elements support the IsKeyEqual
		// method then it is invoked to determine if the elements are
		// equivalent.  If they are they are merged; otherwise they are appended
		// to the slice.  If the elements do not support the IsKeyEqual method
		// then the slice is overwritten if the "OverwriteSlices" transform
		// setting is asserted.
		return func(dst, src reflect.Value) error {
			var isKeyEqual = reflect.Value{}

			if src.IsNil() == true {
				// Assume that the user wants to keep the contents of dst.
				return nil
			} else if src.Len() == 0 {
				// The source is a non-nil empty array.  Assume that the user
				// wants to overwrite the destination array with an empty list.
				dst.Set(src)
				return nil
			} else if dst.IsNil() || dst.Len() == 0 {
				// The destination is nil or has no entries so overwrite the
				// destination with the source.
				dst.Set(src)
				return nil
			} else {
				// Try to merge the two arrays if their elements support the
				// function "IsKeyEqual".
				isKeyEqual = dst.Index(0).MethodByName("IsKeyEqual")
				if !isKeyEqual.IsValid() {
					if t.OverwriteSlices {
						// The elements do not support IsKeyEqual and the caller
						// wants to overwrite unknown slices so overwrite the
						// destination with the contents of source.
						dst.Set(src)
					}
					return nil
				}
			}

			// Otherwise we are going to merge the two slices using the
			// result of IsKeyEqual on each element.
			for i := 0; i < src.Len(); i++ {
				found := false
				for j := 0; j < dst.Len(); j++ {
					isKeyEqual = dst.Index(j).MethodByName("IsKeyEqual")
					result := isKeyEqual.Call([]reflect.Value{src.Index(i)})
					if result[0].Bool() {
						// Individual array elements are equivalent therefore
						// recursively merge them

						// We are working with reflections so we cannot call
						// the mergo.Merge API directly since we do not have
						// direct access to the original variables.
						merge := reflect.ValueOf(mergo.Merge)
						result = merge.Call([]reflect.Value{dst.Index(j).Addr(),
							src.Index(i),
							reflect.ValueOf(mergo.WithOverride),
							reflect.ValueOf(mergo.WithTransformers(t))})
						if result[0].IsValid() && result[0].IsNil() == false {
							return result[0].Interface().(error)
						}

						found = true
						break
					}
				}

				if !found {
					// The source element was not found in the destination array
					// therefore append it to the end.
					dst.Set(reflect.Append(dst, src.Index(i)))
				}
			}

			return nil
		}
	}

	return nil
}
