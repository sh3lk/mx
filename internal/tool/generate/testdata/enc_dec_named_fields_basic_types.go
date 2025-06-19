// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// EXPECTED
// enc.Int((int)(a0))
// enc.String((string)(a1))
// *(*int)(&a0) = dec.Int()
// *(*string)(&a1) = dec.String()
// Preallocate

// UNEXPECTED
// func mx_enc_A
// func mx_dec_A
// func mx_enc_B
// func mx_dec_B
// mx_size_A(x *A)
// mx_size_B(x *B)

// Generate methods for named types that are basic types. Verify that no
// enc/dec methods are generated for the types, and we rely on basic types
// enc/dec instead.
package foo

import (
	"context"

	"github.com/sh3lk/mx"
)

type A int
type B string
type Foo interface{}

type foo interface {
	M(context.Context, A, B) error
}

type impl struct{ mx.Implements[foo] }

func (l *impl) M(context.Context, A, B) error {
	return nil
}
