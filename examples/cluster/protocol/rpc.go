package protocol

type NewUserRequest struct {
	Nickname string `json:"nickname"`
	GateUid  int64  `json:"gateUid"`
}
