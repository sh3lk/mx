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
// Preallocate
// func (x *A) MXMarshal(enc *codegen.Encoder)
// func (x *A) MXUnmarshal(dec *codegen.Decoder)
// func (x *B) MXMarshal(enc *codegen.Encoder)
// func (x *B) MXUnmarshal(dec *codegen.Decoder)
// mx_size_A_7d96e200(x *A)
// mx_size_B_1fd0dae2(x *B)

// Generate methods for named types that are structs.
package foo

import (
	"context"

	"github.com/sh3lk/mx"
)

type B struct {
	mx.AutoMarshal
	A
}

type A struct {
	mx.AutoMarshal
	A1 int
	A2 string
}

type foo interface {
	M(context.Context, A, B) error
}

type impl struct{ mx.Implements[foo] }

func (l *impl) M(context.Context, A, B) error {
	return nil
}
