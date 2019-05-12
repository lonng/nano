// Copyright (c) nano Author. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package nano

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lonng/nano/internal/env"
)

// VERSION returns current nano version
var VERSION = "0.5.0"

var (
	// app represents the current server process
	app = &struct {
		name    string    // current application name
		startAt time.Time // startup time
	}{}
)

// init default configs
func init() {
	// application initialize
	app.name = strings.TrimLeft(filepath.Base(os.Args[0]), "/")
	app.startAt = time.Now()

	// environment initialize
	if wd, err := os.Getwd(); err != nil {
		panic(err)
	} else {
		env.Wd, _ = filepath.Abs(wd)
	}
}
