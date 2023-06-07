package websocket

import (
	"context"
	"crypto/sha1"
	"fmt"
	cmap "github.com/orcaman/concurrent-map/v2"
	"golang.org/x/net/websocket"
	"log"
	"time"
)

var wsOriginConnMap = cmap.New[context.CancelFunc]()

func topicKey(topic string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(topic)))
}

func connect(ctx context.Context, topic string) error {
	// 连接源服务器
	config, err := websocket.NewConfig("wss://ws.okx.com/ws/v5/public", "http://localhost")
	if err != nil {
		return err
	}
	topicKey := topicKey(topic)
	for {
		if ctx.Err() != nil {
			return err
		}
		// 建立连接
		wsConn, err := websocket.DialConfig(config)
		if err != nil {
			log.Printf("try connect origin but error becuse of %s\n", err.Error())
		} else {
			// 发送订阅参数
			err = websocket.Message.Send(wsConn, topic)
			if err != nil {
				log.Printf("subscribe channel error becuse of %s\n", err.Error())
			} else {
				// 开始接收消息
				for {
					var data = AnyMessages{}
					err = AnyMessageCodec.Receive(wsConn, &data)
					if err != nil {
						break
					}
					if data.PayloadType == websocket.TextFrame {
						message := string(data.Msg)
						log.Println(message)
						DispatchNewMessageToUser(topicKey, message)
					} else if data.PayloadType == websocket.BinaryFrame {
						log.Printf("bytes data length %d\n", len(data.Msg))
						DispatchNewMessageToUser(topicKey, data.Msg)
					}
				}
			}
		}
		// 尝试关闭连接
		wsConn.Close()
		// 重试条件判断
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(5 * time.Second):
			// 等待5s后尝试重新连接
			continue
		}
	}
}

func RemoveTopic(topic string) {
	key := topicKey(topic)
	wsOriginConnMap.RemoveCb(key, func(key string, cancelFunc context.CancelFunc, exists bool) bool {
		if exists {
			// 调用cancel，关闭连接
			go cancelFunc()
			return true
		}
		return false
	})
}

func RemoveAllTopic() {
	iterator := wsOriginConnMap.IterBuffered()
	for item := range iterator {
		go item.Val()
	}
	wsOriginConnMap.Clear()
}

func OriginConnExistTopicKey(topicKey string) bool {
	return wsOriginConnMap.Has(topicKey)
}
