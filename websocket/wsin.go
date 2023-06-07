package websocket

import (
	"context"
	"golang.org/x/net/websocket"
	"log"
	"time"
)

// NewWebsocketHandleClosure 处理用户的WS连接请求
func NewWebsocketHandleClosure(ctx context.Context) func(conn *websocket.Conn) {
	return func(conn *websocket.Conn) {
		// 尾调关闭WS连接
		defer conn.Close()
		closeFlag := make(chan any)
		defer close(closeFlag)
		wsConn := NewWsConn(ctx, conn)
		// 注册连接
		go RegisterWsConn(ctx, closeFlag, wsConn)
		// 启动检查线程
		go checkUserIdleConn(ctx, closeFlag, wsConn)
		// 这里等待客户端发送Ping，并hold住连接
		for {
			var anyMessage = AnyMessages{}
			err := AnyMessageCodec.Receive(conn, &anyMessage)
			if err != nil {
				return
			}
			handleReceivedPingMessage(&anyMessage, wsConn)
		}
	}
}

func checkUserIdleConn(ctx context.Context, closeFlag <-chan any, wsConn *WsConn) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-closeFlag:
			return
		case <-ticker.C:
			// 检查空闲连接
			wsConn.tryCloseIdle()
			// 检查源连接
			if !OriginConnExistTopicKey(wsConn.topicKey()) {
				wsConn.closeConn()
			}
		}
	}
}

func handleReceivedPingMessage(userMessage *AnyMessages, wsConn *WsConn) {
	if userMessage.PayloadType == websocket.PingFrame {
		log.Printf("recv ping frame from %s\n", wsConn.id)
		wsConn.resetLastTs()
	} else if userMessage.PayloadType == websocket.TextFrame {
		message := string(userMessage.Msg)
		if message == "ping" {
			log.Printf("recv ping text from %s\n", wsConn.id)
			wsConn.resetLastTs()
		}
	}
}
