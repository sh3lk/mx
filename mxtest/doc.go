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

// Package mxtest provides a way to test MX components.
//
// Use [mxtest.Test] to exercise a set of components. For example,
// imagine we have a Reverser component with a Reverse method that
// reverses strings. To test the Reverser component, we can create a
// reverser_test.go file with the following contents.
//
//	func TestReverse(t *testing.T) {
//	    mxtest.Local.Test(t, func(t *testing.T, reverser Reverser) {
//		got, err := reverser.Reverse(ctx, "diaper drawer")
//		if err != nil {
//		    t.Fatal(err)
//		}
//		if want := "reward repaid"; got != want {
//		    t.Fatalf("got %q, want %q", got, want)
//		}
//	    })
//	}
//
// Multiple Runners are provided by mxtest. The runners differ in
// whether or not components are placed in different processes and
// whether or not RPCs are used for components in the same process.
//
//  1. Local: all components are placed in a single process and method
//     invocations are local procedure calls. This is similar to what
//     happens when you run the MX binary directly or run
//     the application using "go run".
//
//  2. Multi: Every component will be placed in a separate process,
//     similar to what happens when you "mx multi deploy" a
//     MX application.
//
//  3. RPC: Every component will be placed in a same process, but
//     method calls will use RPCs. This mode is most useful when
//     profiling or collecting coverage information.
//
// Example:
//
//	func TestReverseSingle(t *testing.T) {
//	  mxtest.Multi.Test(t, func(t *testing.T, reverser Reverser) {
//	    // ...
//	  })
//	}
package mxtest
