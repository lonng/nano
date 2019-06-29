package chat

import "github.com/lonng/nano/component"

var (
	// All services in master server
	Services = &component.Components{}

	roomService = newRoomService()
)

func init() {
	Services.Register(roomService)
}
