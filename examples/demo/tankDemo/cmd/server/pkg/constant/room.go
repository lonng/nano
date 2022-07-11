package constant

type RoomStatus int32

const (
	// 创建
	RoomStatusCreate RoomStatus = iota
	// 游戏
	RoomStatusPlaying
	// 游戏终/中止
	RoomStatusInterruption
	// 已销毁
	RoomStatusDestory
)

var stringify = [...]string{
	RoomStatusCreate:       "创建",
	RoomStatusPlaying:      "游戏中",
	RoomStatusInterruption: "游戏终/中止",
	RoomStatusDestory:      "已销毁",
}

func (s RoomStatus) String() string {
	return stringify[s]
}
