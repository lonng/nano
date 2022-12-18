package encoding

import (
	"fmt"
	"testing"
)

func TestGetBytes1(t *testing.T) {
	str1 := `{
    "id": 1001,
    "name": "room1001"
}`
	fmt.Printf("%v", []byte(str1))
}
