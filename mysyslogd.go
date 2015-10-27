package main

import (
	"flag"
	"fmt"
	"github.com/ziutek/syslog"
	"os"
	"os/signal"
	"syscall"
)

type handler struct {
	*syslog.BaseHandler
}

func newHandler() *handler {
	h := handler{syslog.NewBaseHandler(5, nil, false)}
	go h.mainLoop()
	return &h
}

func (h *handler) mainLoop() {
	for {
		m := h.Get()
		if m == nil {
			break
		}
		fmt.Println(m)
	}
	h.End()
}

func main() {
	address := flag.String("addr", ":5514", "address")
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
