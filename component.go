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
	"github.com/lonng/nano/component"
)

var (
	comps = make([]regComp, 0)
)

type regComp struct {
	comp component.Component
	opts []component.Option
}

func startupComponents() {
	// component initialize hooks
	for _, c := range comps {
		c.comp.Init()
	}

	// component after initialize hooks
	for _, c := range comps {
		c.comp.AfterInit()
	}

	// register all components
	for _, c := range comps {
		if err := handler.register(c.comp, c.opts); err != nil {
			logger.Println(err.Error())
		}
	}

	handler.DumpServices()
}

func shutdownComponents() {
	// reverse call `BeforeShutdown` hooks
	length := len(comps)
	for i := length - 1; i >= 0; i-- {
		comps[i].comp.BeforeShutdown()
	}

	// reverse call `Shutdown` hooks
	for i := length - 1; i >= 0; i-- {
		comps[i].comp.Shutdown()
	}
}
