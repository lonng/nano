package errno

import "errors"

type ErrCode int32 //错误码

const CodeSuccess int32 = 0

//go:generate stringer -type ErrCode -linecomment
const (
	RoomNotFound        ErrCode = iota + 10000 // 您输入的房间号不存在, 请确认后再次输入
	RoomPlayerNumEnough                        // 您加入的房间已经满人, 请确认房间号后再次确认
	RoomExist                                  // 房间已存在
	RoomJoinFailed                             // 加入房间失败

	RoomPlayerNotFound // 房间找不到该玩家
)

func (i ErrCode) Int32() int32 {
	return int32(i)
}

func (i ErrCode) Error() error {
	return errors.New(i.String())
}
