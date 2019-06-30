package json

import (
	"reflect"
	"testing"
)

type Message struct {
	Code int    `json:"code"`
	Data string `json:"data"`
}

func TestSerializer_Serialize(t *testing.T) {
	m := Message{1, "hello world"}
	s := NewSerializer()
	b, err := s.Marshal(m)
	if err != nil {
		t.Fail()
	}

	m2 := Message{}
	if err := s.Unmarshal(b, &m2); err != nil {
		t.Fail()
	}

	if !reflect.DeepEqual(m, m2) {
		t.Fail()
	}
}

func BenchmarkSerializer_Serialize(b *testing.B) {
	m := &Message{100, "hell world"}
	s := NewSerializer()

	for i := 0; i < b.N; i++ {
		if _, err := s.Marshal(m); err != nil {
			b.Fatalf("unmarshal failed: %v", err)
		}
	}

	b.ReportAllocs()
}

func BenchmarkSerializer_Deserialize(b *testing.B) {
	m := &Message{100, "hell world"}
	s := NewSerializer()

	d, err := s.Marshal(m)
	if err != nil {
		b.Error(err)
	}

	for i := 0; i < b.N; i++ {
		m1 := &Message{}
		if err := s.Unmarshal(d, m1); err != nil {
			b.Fatalf("unmarshal failed: %v", err)
		}
	}
}
