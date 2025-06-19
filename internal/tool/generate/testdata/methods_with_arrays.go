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
// var a0 [9123]X
// var a0 [3][5]int
// var a1 [2][2][2]float64
// var a1 [12]int
// r0, appErr := s.impl.A
// mx_enc_array_9123_X
// mx_enc_array_12_int
// mx_dec_array_2048_string
// mx_enc_array_3_array_5_int
// mx_enc_array_2_array_2_array_2_float64
// mx_size_X_4cd17e8a(x *X)

// UNEXPECTED
// c.Args.Encode
// c.Results.Decode

// Methods with arrays as arguments and results.
package foo

import (
	"context"

	"github.com/sh3lk/mx"
)

const N = 1024

type foo interface {
	A(context.Context, [9123]X, [5 + 7]int) ([2 * N]string, error)
	B(context.Context, [3][5]int, [2][2][2]float64) error
}

type X struct {
	mx.AutoMarshal
	a int
}

type impl struct{ mx.Implements[foo] }

func (l *impl) A(context.Context, [9123]X, [5 + 7]int) ([2 * N]string, error) {
	return [2 * N]string{}, nil
}

func (l *impl) B(context.Context, [3][5]int, [2][2][2]float64) error {
	return nil
}
