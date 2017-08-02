package protobuf

import (
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
)

type Message struct {
	Data *string `protobuf:"bytes,1,name=data"`
}

func (m *Message) Reset()         { *m = Message{} }
func (m *Message) String() string { return proto.CompactTextString(m) }
func (*Message) ProtoMessage()    {}

func TestProtobufSerialezer_Serialize(t *testing.T) {
	m := &Message{proto.String("hello")}
	s := NewSerializer()

	b, err := s.Serialize(m)
	if err != nil {
		t.Error(err)
	}

	m1 := &Message{}
	s.Deserialize(b, m1)

	if !reflect.DeepEqual(m, m1) {
		t.Fail()
	}
}

func BenchmarkSerializer_Serialize(b *testing.B) {
	m := &Message{proto.String("hello")}
	s := NewSerializer()

	for i := 0; i < b.N; i++ {
		s.Serialize(m)
	}

	b.ReportAllocs()
}

func BenchmarkSerializer_Deserialize(b *testing.B) {
	m := &Message{proto.String("hello")}
	s := NewSerializer()

	d, err := s.Serialize(m)
	if err != nil {
		b.Error(err)
	}

	for i := 0; i<b.N; i++ {
		m1 := &Message{}
		s.Deserialize(d, m1)
	}
}