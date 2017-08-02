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
