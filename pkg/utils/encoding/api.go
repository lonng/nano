package encoding

import (
	"github.com/lonng/nano/pkg/utils/unsafex"
)

func MarshalToString(v interface{}) string {
	bts, _ := GetJsonCodec().Marshal(v)
	return unsafex.BytesString(bts)
}

func UnmarshalFromString(str string, v interface{}) error {
	return GetJsonCodec().Unmarshal(unsafex.StringBytes(str), v)
}
