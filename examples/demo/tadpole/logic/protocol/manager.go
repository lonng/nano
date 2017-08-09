package protocol

type JoyLoginRequest struct {
	Username  string `json:"username"`
	Cipher    string `json:"cipher"`
	Timestamp int    `json:"timestamp"`
}

type LoginResponse struct {
	Status int    `json:"status"` //登录状态
	ID     int64  `json:"id"`     //登录成功后ID
	Error  string `json:"error"`  //登录失败的错误消息
}
