package websocket

import (
	"errors"
	"golang.org/x/net/websocket"
)

var emptyMsg = make([]byte, 0)

type AnyMessages struct {
	Msg         []byte
	PayloadType byte
}

var AnyMessageCodec = websocket.Codec{Marshal: websocket.Message.Marshal, Unmarshal: anyUnmarshal}

func anyUnmarshal(msg []byte, payloadType byte, v interface{}) error {
	m, ok := v.(*AnyMessages)
	if !ok {
		return errors.New("not AnyMessages")
	}
	m.PayloadType = payloadType
	m.Msg = msg
	return nil
}
