package msg_svc

import (
	originLog "log"

	jsonx "github.com/lonng/nano/serialize/json"
	test1V1 "github.com/suhanyujie/throw_interface/golang_pb/test1/v1"
)

func SetWorkingForConn() {

}

func DecodePacketData(pData []byte) (test1V1.IRequestProtocol, error) {
	inputData := test1V1.IRequestProtocol{}
	coder := jsonx.NewSerializer()
	err := coder.Unmarshal(pData, inputData)
	if err != nil {
		originLog.Printf("[DecodePacketData] Unmarshal err: %v \n", err)
		return inputData, err
	}

	return inputData, nil
}
