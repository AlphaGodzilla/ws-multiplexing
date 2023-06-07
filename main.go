package main

import (
	"context"
	"golang.org/x/net/websocket"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	websocket2 "ws-multiplexing/websocket"
)

func HttpHandler(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("Welcom"))
	if err != nil {
		return
	}
}

func terminateServer(ctx context.Context, server *http.Server) {
	<-ctx.Done()
	//fmt.Println("server already shutdown")
	err := server.Shutdown(context.TODO())
	if err != nil {
		log.Fatalln(err)
	}
	os.Exit(0)
}

func main() {
	ctx := context.Background()
	ctx, _ = signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGKILL)
	server := &http.Server{
		Addr:    ":8080",
		Handler: http.Handler(websocket.Handler(websocket2.NewWebsocketHandleClosure(ctx))),
	}
	go terminateServer(ctx, server)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalln(err)
	}

	//var wg sync.WaitGroup
	//wg.Add(1)
	//go wsorigin.AddOriginConnIfNeed(ctx, "{\"op\":\"subscribe\",\"args\":[{\"channel\":\"trades\",\"instId\":\"BTC-USDT\"}]}")
	//wg.Wait()
}
