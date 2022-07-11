package tank

type Context struct {
	RoomId uint64
	Uid    int64
}

func (c *Context) Reset() {
}

func (c *Context) String() string {
	return ""
}
