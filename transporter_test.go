package nano

import (
	"reflect"
	"testing"

	"github.com/lonnng/nano/packet"
)

func Test1(t *testing.T) {
	if !reflect.DeepEqual(heartbeatPacket, append([]byte{packet.Heartbeat, 0x00, 0x00, 0x00})) {
		t.Error("wrong heartbeat packet")
	}
}
