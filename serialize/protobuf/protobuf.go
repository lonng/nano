package protobuf

import (
	"errors"

	"github.com/golang/protobuf/proto"
)

var ErrWrongValueType = errors.New("struct must be able to be converted to proto.Message")

type Serializer struct{}

func NewSerializer() *Serializer {
	return &Serializer{}
}

func (s *Serializer) Serialize(v interface{}) ([]byte, error) {
	pb, ok := v.(proto.Message)
	if !ok {
		return nil, ErrWrongValueType
	}
	return proto.Marshal(pb)
}

func (s *Serializer) Deserialize(data []byte, v interface{}) error {
	pb, ok := v.(proto.Message)
	if !ok {
		return ErrWrongValueType
	}
	return proto.Unmarshal(data, pb)
}
