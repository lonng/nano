package packet

import (
	"reflect"
	"testing"
)

func TestPack(t *testing.T) {
	data := []byte("hello world")
	p1 := &Packet{Type: PacketType(1), Data: data, Length: len(data)}
	pp1, err := p1.Pack()
	if err != nil {
		t.Error(err.Error())
	}
	upp1, rest, err := Unpack(pp1)
	if err != nil {
		t.Error(err.Error())
	}
	if len(rest) > 0 {
		t.Error("rest should empty")
	}
	if !reflect.DeepEqual(p1, upp1) {
		t.Fail()
	}

	p2 := &Packet{Type: PacketType(5), Data: data, Length: len(data)}
	pp2, err := p2.Pack()
	if err != nil {
		t.Error(err.Error())
	}
	upp2, rest, err := Unpack(pp2)
	if err != nil {
		t.Error(err.Error())
	}
	if len(rest) > 0 {
		t.Error("rest should empty")
	}
	if !reflect.DeepEqual(p2, upp2) {
		t.Fail()
	}

	p3 := &Packet{Type: PacketType(0), Data: data, Length: len(data)}
	if _, err := p3.Pack(); err == nil {
		t.Error("should err")
	}

	p4 := &Packet{Type: PacketType(6), Data: data, Length: len(data)}
	if _, err := p4.Pack(); err == nil {
		t.Error("should err")
	}

	p5 := &Packet{Type: PacketType(5), Data: data, Length: len(data)}
	pp5, err := p5.Pack()
	if err != nil {
		t.Error(err.Error())
	}
	upp5, rest, err := Unpack(append(pp5, []byte{0x01, 0x04, 0x09, 0xF0}...))
	if err != nil {
		t.Error(err.Error())
	}
	if !reflect.DeepEqual(rest, []byte{0x01, 0x04, 0x09, 0xF0}) {
		t.Error("wrong rest")
	}
	if !reflect.DeepEqual(p5, upp5) {
		t.Fail()
	}
}
