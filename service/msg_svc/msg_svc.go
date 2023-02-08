package msg_svc

import (
	originLog "log"

	jsonx "github.com/lonng/nano/serialize/json"
	throwV1 "github.com/suhanyujie/throw_interface/golang_pb/throw/v1"
)

func SetWorkingForConn() {

}

func DecodePacketData(pData []byte) (throwV1.IRequestProtocol, error) {
	inputData := throwV1.IRequestProtocol{}
	coder := jsonx.NewSerializer()
	err := coder.Unmarshal(pData, inputData)
	if err != nil {
		originLog.Printf("[DecodePacketData] Unmarshal err: %v \n", err)
		return inputData, err
	}

	return inputData, nil
}
