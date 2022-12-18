package unsafex

import (
	"reflect"
	"unsafe"
)

type Pointer = unsafe.Pointer
type Bytes = []byte

//return GoString's buffer slice(enable modify string)
func StringBytes(s string) Bytes {
	var bh reflect.SliceHeader
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh.Data, bh.Len, bh.Cap = sh.Data, sh.Len, sh.Len
	return *(*Bytes)(unsafe.Pointer(&bh))
}

// convert b to string without copy
func BytesString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// returns &s[0], which is not allowed in go
func StringPointer(s string) Pointer {
	p := (*reflect.StringHeader)(unsafe.Pointer(&s))
	return Pointer(p.Data)
}

// returns &b[0], which is not allowed in go
func BytesPointer(b []byte) Pointer {
	p := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	return Pointer(p.Data)
}
