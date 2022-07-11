package errno

type ErrCode int64 //错误码

// 2、定义errorCode
//go:generate stringer -type ErrCode -linecomment
const (
	RoomNotFound        ErrCode = iota + 10000 // "您输入的房间号不存在, 请确认后再次输入"
	RoomPlayerNumEnough                        // "您加入的房间已经满人, 请确认房间号后再次确认"
)
