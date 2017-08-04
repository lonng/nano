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

package component

import (
	"errors"
	"reflect"
)

type Handler struct {
	Receiver reflect.Value  // receiver of method
	Method   reflect.Method // method stub
	Type     reflect.Type   // low-level type of method
	IsRawArg bool           // whether the data need to serialize
}

type Service struct {
	Name     string              // name of service
	Receiver reflect.Value       // receiver of methods for the service
	Type     reflect.Type        // type of the receiver
	Methods  map[string]*Handler // registered methods
}

// Register publishes in the service the set of methods of the
// receiver value that satisfy the following conditions:
// - exported method of exported type
// - two arguments, both of exported type
// - the first argument is *session.Session
// - the second argument is []byte or a pointer
func (s *Service) ScanHandler() error {
	if s.Name == "" {
		return errors.New("handler.Register: no service name for type " + s.Type.String())
	}
	if !isExported(s.Name) {
		return errors.New("handler.Register: type " + s.Name + " is not exported")
	}

	// Install the methods
	s.Methods = suitableHandlerMethods(s.Type)

	if len(s.Methods) == 0 {
		str := ""
		// To help the user, see if a pointer receiver would work.
		method := suitableHandlerMethods(reflect.PtrTo(s.Type))
		if len(method) != 0 {
			str = "handler.Register: type " + s.Name + " has no exported methods of suitable type (hint: pass a pointer to value of that type)"
		} else {
			str = "handler.Register: type " + s.Name + " has no exported methods of suitable type"
		}
		return errors.New(str)
	}

	for i := range s.Methods {
		s.Methods[i].Receiver = s.Receiver
	}

	return nil
}
