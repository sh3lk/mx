// Copyright 2023 Google LLC
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

// testprogram is used by bin tests.
package main

import (
	"context"

	"github.com/sh3lk/mx"
)

//go:generate ../../../cmd/mx/mx generate

type A interface{}
type B interface{}
type C interface{}

type app struct {
	mx.Implements[mx.Main]
	a      mx.Ref[A]   //lint:ignore U1000 intentionally declared but not used
	appLis mx.Listener //lint:ignore U1000 intentionally declared but not used
}

func (*app) Main(context.Context) error { return nil }

type a struct {
	mx.Implements[A]
	b            mx.Ref[B]   //lint:ignore U1000 intentionally declared but not used
	c            mx.Ref[C]   //lint:ignore U1000 intentionally declared but not used
	aLis1, aLis2 mx.Listener //lint:ignore U1000 intentionally declared but not used
	unused       mx.Listener `mx:"aLis3"` //lint:ignore U1000 intentionally declared but not used
}

type b struct {
	mx.Listener
	mx.Implements[B]
}

type c struct {
	mx.Listener `mx:"cLis"`
	mx.Implements[C]
}

func main() {}
