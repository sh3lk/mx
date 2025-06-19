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
// mx_size_ptr_ptr_ptr_A
// mx_size_ptr_ptr_A
// mx_size_ptr_A
// mx_size_A
// mx_size_ptr_ptr_ptr_B
// mx_size_ptr_ptr_B
// mx_size_ptr_B
// mx_size_B
// mx_size_ptr_ptr_ptr_int
// mx_size_ptr_ptr_int
// mx_size_ptr_int

// Deeply nested pointers and structs.
package foo

import (
	"context"

	"github.com/sh3lk/mx"
)

type A struct {
	mx.AutoMarshal
	b ***B
}

type B struct {
	mx.AutoMarshal
	x ***int
}

type foo interface {
	M(context.Context, ***A) error
}

type impl struct{ mx.Implements[foo] }

func (l *impl) M(context.Context, ***A) error {
	return nil
}
