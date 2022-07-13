package game

import (
	"encoding/base64"
	"fmt"
	"log"

	"github.com/lonng/nano/pipeline"
	"github.com/lonng/nano/session"
	"github.com/xxtea/xxtea-go/xxtea"
)

var xxteaKey = []byte("7AEC4MA152BQE9HWQ7KB")

type Crypto struct {
	key []byte
}

func newCrypto() *Crypto {
	return &Crypto{xxteaKey}
}

func (c *Crypto) inbound(s *session.Session, msg *pipeline.Message) error {
	out, err := base64.StdEncoding.DecodeString(string(msg.Data))
	if err != nil {
		log.Printf("Inbound Error=%s, In=%s", err.Error(), string(msg.Data))
		return err
	}

	out = xxtea.Decrypt(out, c.key)
	if out == nil {
		return fmt.Errorf("decrypt error=%s", err.Error())
	}
	msg.Data = out
	return nil
}

func (c *Crypto) outbound(s *session.Session, msg *pipeline.Message) error {
	out := xxtea.Encrypt(msg.Data, c.key)
	msg.Data = []byte(base64.StdEncoding.EncodeToString(out))
	return nil
}
