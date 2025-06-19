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
// func (x *XYZ) MXMarshal(enc *codegen.Encoder)
// func (x *XYZ) MXUnmarshal(dec *codegen.Decoder)
// func (x *XY) MXMarshal(enc *codegen.Encoder)
// func (x *XY) MXUnmarshal(dec *codegen.Decoder)
// func (x *X) MXMarshal(enc *codegen.Encoder)
// func (x *X) MXUnmarshal(dec *codegen.Decoder)
// func (x *Z) MXMarshal(enc *codegen.Encoder)
// func (x *Z) MXUnmarshal(dec *codegen.Decoder)
// func (x *XYZ) MXMarshal(enc *codegen.Encoder)
// func (x *XYZ) MXUnmarshal(dec *codegen.Decoder)
// func (x *XY) MXMarshal(enc *codegen.Encoder)
// func (x *XY) MXUnmarshal(dec *codegen.Decoder)
// func (x *X) MXMarshal(enc *codegen.Encoder)
// func (x *X) MXUnmarshal(dec *codegen.Decoder)
// func (x *Z) MXMarshal(enc *codegen.Encoder)
// func (x *Z) MXUnmarshal(dec *codegen.Decoder)
// func (x *Y) MXMarshal(enc *codegen.Encoder)
// func (x *Y) MXUnmarshal(dec *codegen.Decoder)
// func (x *Y) MXMarshal(enc *codegen.Encoder)
// func (x *Y) MXUnmarshal(dec *codegen.Decoder)
// func (x *W) MXMarshal(enc *codegen.Encoder)
// func (x *W) MXUnmarshal(dec *codegen.Decoder)
// func (x *W) MXMarshal(enc *codegen.Encoder)
// func (x *W) MXUnmarshal(dec *codegen.Decoder)
// EncodeBinaryMarshaler
// DecodeBinaryUnmarshaler

// UNEXPECTED
// Preallocate

// Generate methods for nested structs. Verify that for structs that have
// all types in the same package or that have custom (Un)marshalBinary methods,
// enc/dec methods are generated.
package foo

import (
	"context"
	"time"

	"github.com/sh3lk/mx"
)

type foo interface {
	M(ctx context.Context, x X, y Y, z Z, w W) error
}

type impl struct{ mx.Implements[foo] }

func (l *impl) M(ctx context.Context, x X, y Y, z Z, w W) error {
	return nil
}

type X struct {
	mx.AutoMarshal
	A1 XY
}

type XY struct {
	mx.AutoMarshal
	A1 XYZ
	B1 string
}

type XYZ struct {
	mx.AutoMarshal
	A1 int64
	A2 string
}

type Y struct {
	mx.AutoMarshal
	ID    int64
	Label string
	When  time.Time
	Text  string
}

type Z struct {
	mx.AutoMarshal
	id string
}

type W struct {
	mx.AutoMarshal
	id time.Time
}
