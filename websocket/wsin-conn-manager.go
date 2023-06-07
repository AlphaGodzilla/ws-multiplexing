package websocket

import (
	"context"
	"crypto/sha1"
	"fmt"
	"github.com/google/uuid"
	cmap "github.com/orcaman/concurrent-map/v2"
	"golang.org/x/net/websocket"
	"log"
	"strings"
	"time"
)

var wsConnRegisterMap = cmap.New[*SameTopicWsConn]()

type SameTopicWsConn struct {
	topic     string
	wsConnMap *cmap.ConcurrentMap[string, *WsConn]
}

type WsConn struct {
	// 连接ID
	id string
	// 订阅参数
	topic string
	// 连接本体
	conn *websocket.Conn
	// 活跃时间戳
	lastTs uint64
}

func (w *WsConn) topicKey() string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(w.topic)))
}

func (w *WsConn) tryCloseIdle() {
	nowTs := time.Now().UnixMilli()
	idleDuration := nowTs - int64(w.lastTs)
	if idleDuration > time.Minute.Milliseconds() {
		// 超过1min没有来往数据则认为是空闲连接，服务端需要主动关闭
		w.closeConn()
	}
}

func (w *WsConn) closeConn() bool {
	err := w.conn.Close()
	if err != nil {
		log.Printf("close idle user conn error becuse of %s\n", err.Error())
	}
	return true
}

func (w *WsConn) resetLastTs() {
	w.lastTs = uint64(time.Now().UnixMilli())
}

func (w *WsConn) sendMessage(message interface{}) error {
	return websocket.Message.Send(w.conn, message)
}

func NewWsConn(ctx context.Context, conn *websocket.Conn) *WsConn {
	// 获取订阅参数
	topic := conn.Request().RequestURI
	id := uuid.NewString()
	return &WsConn{
		id:     id,
		topic:  topic,
		conn:   conn,
		lastTs: uint64(time.Now().UnixMilli()),
	}
}

func RegisterWsConn(ctx context.Context, closeFlag <-chan any, wsConn *WsConn) {
	addConn(ctx, wsConn)
	defer unregisterWsConn(wsConn)
	<-closeFlag
}

func addConn(ctx context.Context, wsConn *WsConn) {
	topicKey := wsConn.topicKey()
	wsConnRegisterMap.Upsert(
		topicKey,
		nil,
		func(exist bool, valueInMap *SameTopicWsConn, newValue *SameTopicWsConn) *SameTopicWsConn {
			if exist {
				return valueInMap
			} else {
				wsConnMap := cmap.New[*WsConn]()
				return &SameTopicWsConn{
					topic:     wsConn.topic,
					wsConnMap: &wsConnMap,
				}
			}
		}).wsConnMap.Upsert(
		wsConn.id,
		wsConn,
		func(exist bool, valueInMap *WsConn, newValue *WsConn) *WsConn {
			if exist {
				return valueInMap
			} else {
				// 这里通知wsOrigin建立与源的连接
				AddOriginConnIfNeed(ctx, strings.Clone(newValue.topic))
				return newValue
			}
		})
}

func unregisterWsConn(wsConn *WsConn) {
	topicKey := wsConn.topicKey()
	wsConnRegisterMap.RemoveCb(topicKey, func(key string, v *SameTopicWsConn, exists bool) bool {
		if exists {
			removed := v.wsConnMap.RemoveCb(wsConn.id, func(key string, v *WsConn, exists bool) bool {
				if exists {
					return true
				} else {
					return false
				}
			})
			return removed && v.wsConnMap.IsEmpty()
		} else {
			return false
		}
	})
}

func UserConnExistTopicKey(topicKey string) bool {
	return wsConnRegisterMap.Has(topicKey)
}

func DispatchNewMessageToUser(topicKey string, message interface{}) {
	value, ok := wsConnRegisterMap.Get(topicKey)
	if !ok {
		return
	}
	iterator := value.wsConnMap.IterBuffered()
	for item := range iterator {
		wsConn := item.Val
		go func() {
			err := wsConn.sendMessage(message)
			if err != nil {
				log.Printf("send new message to user error becuse of %s\n", err.Error())
			}
		}()
	}
}
