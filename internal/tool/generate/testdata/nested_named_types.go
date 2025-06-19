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
// func (x *A) MXMarshal(enc *codegen.Encoder)
// func (x *A) MXUnmarshal(dec *codegen.Decoder)
// func (x *B) MXMarshal(enc *codegen.Encoder)
// func (x *B) MXUnmarshal(dec *codegen.Decoder)
// func (x *C) MXMarshal(enc *codegen.Encoder)
// func (x *C) MXUnmarshal(dec *codegen.Decoder)

// Nested named types.
package foo

import (
	"context"

	"github.com/sh3lk/mx"
)

type A struct {
	mx.AutoMarshal
	B
}

type B struct {
	mx.AutoMarshal
	C
}

type C struct {
	mx.AutoMarshal
	x int
}

type foo interface {
	MethodOne(context.Context, A) error
	MethodTwo(context.Context, B) error
}

type impl struct{ mx.Implements[foo] }

func (l *impl) MethodOne(context.Context, A) error {
	return nil
}

func (l *impl) MethodTwo(context.Context, B) error {
	return nil
}
