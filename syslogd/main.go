package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ziutek/syslog"
)

func newHandler() *syslog.BaseHandler {
	h := syslog.NewBaseHandler(5, nil, false)
	go func() {
		defer h.End()
		for {
			m := h.Get()
			if m == nil {
				break
			}
			fmt.Println(m)
		}
	}()

	return h
}

func main() {
	address := flag.String("addr", ":514", "address")
	flag.Parse()

	server := syslog.NewServer()
	server.AddHandler(newHandler())
	server.Listen(*address)

	sig := make(chan os.Signal, 2)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	<-sig

	server.Shutdown()
	fmt.Println("Server is now down.")
}
