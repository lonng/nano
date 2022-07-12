package tank

type Context struct {
	Uid    int64
	RoomId uint64
}

func (c *Context) Reset() {
	c.RoomId = 0
}
