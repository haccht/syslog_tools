package main

import (
	"flag"
	"log"
	"log/syslog"
)

func main() {
	address := flag.String("addr", ":5514", "syslog server")
	level := flag.String("level", "info", "syslog level")
	message := flag.String("msg", "", "syslog message")
	flag.Parse()

	logWriter, err := syslog.Dial("udp", *address, syslog.LOG_ERR, "logger")
	defer logWriter.Close()
	if err != nil {
		log.Fatal("error")
	}

	switch *level {
	case "emerg":
		logWriter.Emerg(*message)
	case "alert":
		logWriter.Alert(*message)
	case "crit":
		logWriter.Crit(*message)
	case "err":
		logWriter.Err(*message)
	case "warning":
		logWriter.Warning(*message)
	case "notice":
		logWriter.Notice(*message)
	case "info":
		logWriter.Info(*message)
	case "debug":
		logWriter.Debug(*message)
	}
}
