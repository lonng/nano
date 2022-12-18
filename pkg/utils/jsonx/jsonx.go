package jsonx

import "encoding/json"

// ToJsonIgnoreErr 便捷的 json 序列化方式，但是效率低！
func ToJsonIgnoreErr(v interface{}) string {
	bArr, _ := json.Marshal(v)
	return string(bArr)
}

func ToJson(v interface{}) (string, error) {
	bArr, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(bArr), nil
}

func FromJson(jsonStr string, o interface{}) error {
	err := json.Unmarshal([]byte(jsonStr), &o)
	if err != nil {
		return err
	}
	return nil
}
