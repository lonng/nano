package protocol

// UpdateMessage contains latest position information of tadpole
type UpdateMessage struct {
	ID       int     `json:"id"`
	X        float32 `json:"x"`
	Y        float32 `json:"y"`
	Name     string  `json:"name"`
	Momentum float32 `json:"momentum"`
	Angle    float32 `json:"angle"`
}

// EnterWorldResponse indicates a new tadpole enter current scene
type EnterWorldResponse struct {
	ID int64 `json:"id"`
}

// LeaveWorldResponse indicates tadpole leave current scene
type LeaveWorldResponse struct {
	ID int64 `json:"id"`
}

// WorldMessage represent a guest message
type WorldMessage struct {
	ID      int64  `json:"id"`
	Message string `json:"message"`
}
