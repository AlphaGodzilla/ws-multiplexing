package websocket

import (
	"context"
	"crypto/sha1"
	"fmt"
	"log"
	"time"
)

func AddOriginConnIfNeed(parentCtx context.Context, topic string) {
	key := fmt.Sprintf("%x", sha1.Sum([]byte(topic)))
	ctx, cancel := context.WithCancel(parentCtx)
	absent := wsOriginConnMap.SetIfAbsent(key, cancel)
	if absent {
		// 启动源连接线程
		go func() {
			err := connect(ctx, topic)
			if err != nil {
				// 运行到这里说明连接线程已经中断，保留topic已无意义，故需移除失活的topic
				RemoveTopic(topic)
				log.Printf("origin connect error becuse of %s\n", err.Error())
			}
		}()
		// 启动检测线程
		go checkOriginIdleConn(ctx, topic)
	}
}

func checkOriginIdleConn(ctx context.Context, topic string) {
	key := topicKey(topic)
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !OriginConnExistTopicKey(key) {
				return
			}
			if !UserConnExistTopicKey(key) {
				// 说明没有监听此topic的用户连接了，则关闭该源连接以节省资源
				RemoveTopic(topic)
			}
		}
	}
}
