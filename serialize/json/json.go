package json

import "encoding/json"

type Serializer struct{}

func NewSerializer() *Serializer {
	return &Serializer{}
}

func (s *Serializer) Serialize(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (s *Serializer) Deserialize(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
