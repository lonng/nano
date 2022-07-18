package tank

import (
	"github.com/lonng/nano/examples/demo/tankDemo/pb"
)

// 帧服务
type lockstep struct {
	frames     map[uint32]*frameData
	frameCount uint32
}

func newLockstep() *lockstep {
	l := &lockstep{
		frames: make(map[uint32]*frameData),
	}

	return l
}

// 帧服务数据
type frameData struct {
	idx  uint32
	cmds []*pb.Input_Notify
}

func newFrameData(index uint32) *frameData {
	f := &frameData{
		idx:  index,
		cmds: make([]*pb.Input_Notify, 0),
	}

	return f
}

func (l *lockstep) reset() {
	l.frames = make(map[uint32]*frameData)
	l.frameCount = 0
}

func (l *lockstep) getFrameCount() uint32 {
	return l.frameCount
}

func (l *lockstep) pushCmd(data *pb.Input_Notify) bool {
	f, ok := l.frames[l.frameCount]
	if !ok {
		f = newFrameData(l.frameCount)
		l.frames[l.frameCount] = f
	}

	// 检查是否同一帧发来两次操作
	//for _, v := range f.cmds {
	//	if v.PlayerID == cmd.PlayerID {
	//		return false
	//	}
	//}

	f.cmds = append(f.cmds, data)

	return true
}

func (l *lockstep) tick() uint32 {
	l.frameCount++
	return l.frameCount
}

func (l *lockstep) getRangeFrames(from, to uint32) []*frameData {
	ret := make([]*frameData, 0, to-from)

	for ; from <= to && from <= l.frameCount; from++ {
		f, ok := l.frames[from]
		if !ok {
			continue
		}
		ret = append(ret, f)
	}

	return ret
}

func (l *lockstep) getFrame(idx uint32) *frameData {

	return l.frames[idx]
}
