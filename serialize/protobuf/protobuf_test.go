package protobuf

import (
	"reflect"
	"testing"

	"github.com/lonnng/nano/benchmark/testdata"
)

func TestProtobufSerialezer_Serialize(t *testing.T) {
	m := &testdata.Ping{Content: "hello"}
	s := NewSerializer()

	b, err := s.Marshal(m)
	if err != nil {
		t.Error(err)
	}

	m1 := &testdata.Ping{}
	s.Deserialize(b, m1)

	if !reflect.DeepEqual(m, m1) {
		t.Fail()
	}
}

func BenchmarkSerializer_Serialize(b *testing.B) {
	m := &testdata.Ping{Content: "hello"}
	s := NewSerializer()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.Marshal(m)
	}
}

func BenchmarkSerializer_Deserialize(b *testing.B) {
	m := &testdata.Ping{Content: "hello"}
	s := NewSerializer()

	d, err := s.Marshal(m)
	if err != nil {
		b.Error(err)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m1 := &testdata.Ping{}
		s.Deserialize(d, m1)
	}
}
