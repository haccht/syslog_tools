package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	flags "github.com/jessevdk/go-flags"
	syslog "github.com/racksec/srslog"
)

func levelPriority(level string) (syslog.Priority, error) {
	level = strings.ToUpper(level)
	switch level {
	case "EMERG":
		return syslog.LOG_EMERG, nil
	case "ALERT":
		return syslog.LOG_ALERT, nil
	case "CRIT":
		return syslog.LOG_CRIT, nil
	case "ERR":
		return syslog.LOG_ERR, nil
	case "WARN", "WARNING":
		return syslog.LOG_WARNING, nil
	case "NOTICE":
		return syslog.LOG_NOTICE, nil
	case "INFO":
		return syslog.LOG_INFO, nil
	case "DEBUG":
		return syslog.LOG_DEBUG, nil
	default:
		return 0, fmt.Errorf("invalid syslog level: %s", level)
	}
}

func facilityPriority(facility string) (syslog.Priority, error) {
	facility = strings.ToUpper(facility)
	switch facility {
	case "KERN":
		return syslog.LOG_KERN, nil
	case "USER":
		return syslog.LOG_USER, nil
	case "MAIL":
		return syslog.LOG_MAIL, nil
	case "DAEMON":
		return syslog.LOG_DAEMON, nil
	case "AUTH":
		return syslog.LOG_AUTH, nil
	case "SYSLOG":
		return syslog.LOG_SYSLOG, nil
	case "LPR":
		return syslog.LOG_LPR, nil
	case "NEWS":
		return syslog.LOG_NEWS, nil
	case "UUCP":
		return syslog.LOG_UUCP, nil
	case "CRON":
		return syslog.LOG_CRON, nil
	case "AUTHPRIV":
		return syslog.LOG_AUTHPRIV, nil
	case "FTP":
		return syslog.LOG_FTP, nil
	case "LOCAL0":
		return syslog.LOG_LOCAL0, nil
	case "LOCAL1":
		return syslog.LOG_LOCAL1, nil
	case "LOCAL2":
		return syslog.LOG_LOCAL2, nil
	case "LOCAL3":
		return syslog.LOG_LOCAL3, nil
	case "LOCAL4":
		return syslog.LOG_LOCAL4, nil
	case "LOCAL5":
		return syslog.LOG_LOCAL5, nil
	case "LOCAL6":
		return syslog.LOG_LOCAL6, nil
	case "LOCAL7":
		return syslog.LOG_LOCAL7, nil
	default:
		return 0, fmt.Errorf("invalid syslog facility: %s", facility)
	}
}

func parsePriority(priority string) (syslog.Priority, error) {
	tokens := strings.Split(priority, ".")

	facility, err := facilityPriority(tokens[0])
	if err != nil {
		return 0, err
	}

	level, err := levelPriority(tokens[1])
	if err != nil {
		return 0, err
	}

	return facility | level, nil
}

func main() {
	var opts struct {
		Connection string `short:"c" long:"network" description:"Connect to this network" choice:"tcp" choice:"udp" default:"udp"`
		Address    string `short:"n" long:"address" description:"Write to this remote syslog server" default:":514"`
		Priority   string `short:"p" long:"priority" description:"Mark given message with this priority" default:"user.notice"`
		Tag        string `short:"t" long:"tag" description:"Mark every line with this tag (default: $0)"`
		Hostname   string `short:"l" long:"hostname" description:"Override syslog sender with this name (default: hostname)"`
	}

	args, err := flags.Parse(&opts)
	if err != nil {
		if fe, ok := err.(*flags.Error); ok && fe.Type == flags.ErrHelp {
			os.Exit(0)
		}
		log.Fatal(err)
	}

	if opts.Tag == "" {
		opts.Tag = os.Args[0]
	}

	if opts.Hostname == "" {
		if hostname, err := os.Hostname(); err == nil {
			opts.Hostname = hostname
		}
	}

	priority, err := parsePriority(opts.Priority)
	if err != nil {
		log.Fatal(err)
	}

	w, err := syslog.Dial(opts.Connection, opts.Address, priority, opts.Tag)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	w.SetHostname(opts.Hostname)
	defer w.Close()

	message := strings.Join(args, " ")
	if len(message) > 0 {
		w.Write([]byte(message))
	}
}
