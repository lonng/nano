package protocol

// JoyLoginRequest represent a login request
type JoyLoginRequest struct {
	Username  string `json:"username"`
	Cipher    string `json:"cipher"`
	Timestamp int    `json:"timestamp"`
}

// LoginResponse represent a login response
type LoginResponse struct {
	Status int    `json:"status"`
	ID     int64  `json:"id"`
	Error  string `json:"error"`
}
