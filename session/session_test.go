package session

import "testing"

func TestNewSession(t *testing.T) {
	s := New(nil)
	if s.ID() < 1 {
		t.Fail()
	}
}

func TestSession_Bind(t *testing.T) {
	s := New(nil)
	uids := []int64{100, 1000, 10000000}
	for i, uid := range uids {
		s.Bind(uid)
		if s.UID() != uids[i] {
			t.Fail()
		}
	}
}

func TestSession_HasKey(t *testing.T) {
	s := New(nil)
	key := "hello"
	value := "world"
	s.Set(key, value)
	if !s.HasKey(key) {
		t.Fail()
	}
}

func TestSession_Float32(t *testing.T) {
	s := New(nil)
	key := "hello"
	value := float32(1.2000)
	s.Set(key, value)
	if value != s.Float32(key) {
		t.Fail()
	}
}

func TestSession_Float64(t *testing.T) {
	s := New(nil)
	key := "hello"
	value := 1.2000
	s.Set(key, value)
	if value != s.Float64(key) {
		t.Fail()
	}
}

func TestSession_Int(t *testing.T) {
	s := New(nil)
	key := "testkey"
	value := 234
	s.Set(key, value)
	if value != s.Int(key) {
		t.Fail()
	}
}

func TestSession_Int8(t *testing.T) {
	s := New(nil)
	key := "testkey"
	value := int8(123)
	s.Set(key, value)
	if value != s.Int8(key) {
		t.Fail()
	}
}

func TestSession_Int16(t *testing.T) {
	s := New(nil)
	key := "testkey"
	value := int16(3245)
	s.Set(key, value)
	if value != s.Int16(key) {
		t.Fail()
	}
}

func TestSession_Int32(t *testing.T) {
	s := New(nil)
	key := "testkey"
	value := int32(5454)
	s.Set(key, value)
	if value != s.Int32(key) {
		t.Fail()
	}
}

func TestSession_Int64(t *testing.T) {
	s := New(nil)
	key := "testkey"
	value := int64(444454)
	s.Set(key, value)
	if value != s.Int64(key) {
		t.Fail()
	}
}

func TestSession_Uint(t *testing.T) {
	s := New(nil)
	key := "testkey"
	value := uint(24254)
	s.Set(key, value)
	if value != s.Uint(key) {
		t.Fail()
	}
}

func TestSession_Uint8(t *testing.T) {
	s := New(nil)
	key := "testkey"
	value := uint8(34)
	s.Set(key, value)
	if value != s.Uint8(key) {
		t.Fail()
	}
}

func TestSession_Uint16(t *testing.T) {
	s := New(nil)
	key := "testkey"
	value := uint16(4645)
	s.Set(key, value)
	if value != s.Uint16(key) {
		t.Fail()
	}
}

func TestSession_Uint32(t *testing.T) {
	s := New(nil)
	key := "testkey"
	value := uint32(12365)
	s.Set(key, value)
	if value != s.Uint32(key) {
		t.Fail()
	}
}

func TestSession_Uint64(t *testing.T) {
	s := New(nil)
	key := "testkey"
	value := uint64(1000)
	s.Set(key, value)
	if value != s.Uint64(key) {
		t.Fail()
	}
}

func TestSession_State(t *testing.T) {
	s := New(nil)
	key := "testkey"
	value := uint64(1000)
	s.Set(key, value)
	state := s.State()
	if value != state[key].(uint64) {
		t.Fail()
	}
}

func TestSession_Restore(t *testing.T) {
	s := New(nil)
	s2 := New(nil)
	key := "testkey"
	value := uint64(1000)
	s.Set(key, value)
	state := s.State()
	s2.Restore(state)
	if value != s2.Uint64(key) {
		t.Fail()
	}
}
