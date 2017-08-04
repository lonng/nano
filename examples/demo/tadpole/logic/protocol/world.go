package protocol

type UpdateMessage struct {
	ID       int     `json:"id"`
	X        float32 `json:"x"`
	Y        float32 `json:"y"`
	Name     string  `json:"name"`
	Momentum float32 `json:"momentum"`
	Angle    float32 `json:"angle"`
}

type EnterWorldResponse struct {
	ID int64 `json:"id"`
}

type LeaveWorldResponse struct {
	ID int64 `json:"id"`
}

type WorldMessage struct {
	ID      int64  `json:"id"`
	Message string `json:"message"`
}
