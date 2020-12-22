package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	flags "github.com/jessevdk/go-flags"
)

// The Priority is a combination of the syslog facility and
// severity. For example, LOG_ALERT | LOG_FTP sends an alert severity
// message from the FTP facility. The default severity is LOG_EMERG;
// the default facility is LOG_KERN.
type Priority int

const severityMask = 0x07
const facilityMask = 0xf8

const (
	// Severity.

	// From /usr/include/sys/syslog.h.
	// These are the same on Linux, BSD, and OS X.
	LOG_EMERG Priority = iota
	LOG_ALERT
	LOG_CRIT
	LOG_ERR
	LOG_WARNING
	LOG_NOTICE
	LOG_INFO
	LOG_DEBUG
)

const (
	// Facility.

	// From /usr/include/sys/syslog.h.
	// These are the same up to LOG_FTP on Linux, BSD, and OS X.
	LOG_KERN Priority = iota << 3
	LOG_USER
	LOG_MAIL
	LOG_DAEMON
	LOG_AUTH
	LOG_SYSLOG
	LOG_LPR
	LOG_NEWS
	LOG_UUCP
	LOG_CRON
	LOG_AUTHPRIV
	LOG_FTP
	_ // unused
	_ // unused
	_ // unused
	_ // unused
	LOG_LOCAL0
	LOG_LOCAL1
	LOG_LOCAL2
	LOG_LOCAL3
	LOG_LOCAL4
	LOG_LOCAL5
	LOG_LOCAL6
	LOG_LOCAL7
)

// A Writer is a connection to a syslog server.
type Writer struct {
	priority Priority
	tag      string
	hostname string
	network  string
	raddr    string

	mu   sync.Mutex // guards conn
	conn net.Conn
}

// Dial establishes a connection to a log daemon by connecting to
// address raddr on the specified network. Each write to the returned
// writer sends a log message with the facility and severity
// (from priority) and tag. If tag is empty, the os.Args[0] is used.
// If network is empty, Dial will connect to the local syslog server.
// Otherwise, see the documentation for net.Dial for valid values
// of network and raddr.
func Dial(network, raddr string, priority Priority, tag, hostname string) (*Writer, error) {
	if priority < 0 || priority > LOG_LOCAL7|LOG_DEBUG {
		return nil, errors.New("log/syslog: invalid priority")
	}

	if tag == "" {
		tag = os.Args[0]
	}

	if hostname == "" {
		hostname, _ = os.Hostname()
	}

	w := &Writer{
		priority: priority,
		tag:      tag,
		hostname: hostname,
		network:  network,
		raddr:    raddr,
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	err := w.connect()
	if err != nil {
		return nil, err
	}
	return w, err
}

// connect makes a connection to the syslog server.
// It must be called with w.mu held.
func (w *Writer) connect() (err error) {
	if w.conn != nil {
		// ignore err from close, it makes sense to continue anyway
		w.Close()
	}

	w.conn, err = net.Dial(w.network, w.raddr)
	if err == nil && w.hostname == "" {
		w.hostname = w.conn.LocalAddr().String()
	}

	return err
}

// Write sends a log message to the syslog daemon.
func (w *Writer) Write(b []byte) (int, error) {
	return w.writeAndRetry(w.priority, string(b))
}

// Close closes a connection to the syslog daemon.
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.conn != nil {
		err := w.conn.Close()
		w.conn = nil
		return err
	}
	return nil
}

// Emerg logs a message with severity LOG_EMERG, ignoring the severity
// passed to New.
func (w *Writer) Emerg(m string) error {
	_, err := w.writeAndRetry(LOG_EMERG, m)
	return err
}

// Alert logs a message with severity LOG_ALERT, ignoring the severity
// passed to New.
func (w *Writer) Alert(m string) error {
	_, err := w.writeAndRetry(LOG_ALERT, m)
	return err
}

// Crit logs a message with severity LOG_CRIT, ignoring the severity
// passed to New.
func (w *Writer) Crit(m string) error {
	_, err := w.writeAndRetry(LOG_CRIT, m)
	return err
}

// Err logs a message with severity LOG_ERR, ignoring the severity
// passed to New.
func (w *Writer) Err(m string) error {
	_, err := w.writeAndRetry(LOG_ERR, m)
	return err
}

// Warning logs a message with severity LOG_WARNING, ignoring the
// severity passed to New.
func (w *Writer) Warning(m string) error {
	_, err := w.writeAndRetry(LOG_WARNING, m)
	return err
}

// Notice logs a message with severity LOG_NOTICE, ignoring the
// severity passed to New.
func (w *Writer) Notice(m string) error {
	_, err := w.writeAndRetry(LOG_NOTICE, m)
	return err
}

// Info logs a message with severity LOG_INFO, ignoring the severity
// passed to New.
func (w *Writer) Info(m string) error {
	_, err := w.writeAndRetry(LOG_INFO, m)
	return err
}

// Debug logs a message with severity LOG_DEBUG, ignoring the severity
// passed to New.
func (w *Writer) Debug(m string) error {
	_, err := w.writeAndRetry(LOG_DEBUG, m)
	return err
}

func (w *Writer) writeAndRetry(p Priority, s string) (int, error) {
	pr := (w.priority & facilityMask) | (p & severityMask)

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.conn != nil {
		if n, err := w.write(pr, s); err == nil {
			return n, err
		}
	}
	if err := w.connect(); err != nil {
		return 0, err
	}
	return w.write(pr, s)
}

// write generates and writes a syslog formatted string. The
// format is as follows: <PRI>TIMESTAMP HOSTNAME TAG[PID]: MSG
func (w *Writer) write(p Priority, msg string) (int, error) {
	// ensure it ends in a \n
	nl := ""
	if !strings.HasSuffix(msg, "\n") {
		nl = "\n"
	}

	err := w.writeString(p, w.hostname, w.tag, msg, nl)
	if err != nil {
		return 0, err
	}
	// Note: return the length of the input, not the number of
	// bytes printed by Fprintf, because this must behave like
	// an io.Writer.
	return len(msg), nil
}

func (w *Writer) writeString(p Priority, hostname, tag, msg, nl string) error {
	timestamp := time.Now().Format(time.RFC3339)
	_, err := fmt.Fprintf(w.conn, "<%d>%s %s %s[%d]: %s%s",
		p, timestamp, hostname,
		tag, os.Getpid(), msg, nl)
	return err
}

// levelPriority converts a level string into
// an appropriate priority level or returns an error
func levelPriority(level string) (Priority, error) {
	level = strings.ToUpper(level)
	switch level {
	case "EMERG":
		return LOG_EMERG, nil
	case "ALERT":
		return LOG_ALERT, nil
	case "CRIT":
		return LOG_CRIT, nil
	case "ERR":
		return LOG_ERR, nil
	case "WARN", "WARNING":
		return LOG_WARNING, nil
	case "NOTICE":
		return LOG_NOTICE, nil
	case "INFO":
		return LOG_INFO, nil
	case "DEBUG":
		return LOG_DEBUG, nil
	default:
		return 0, fmt.Errorf("invalid syslog level: %s", level)
	}
}

// facilityPriority converts a facility string into
// an appropriate priority level or returns an error
func facilityPriority(facility string) (Priority, error) {
	facility = strings.ToUpper(facility)
	switch facility {
	case "KERN":
		return LOG_KERN, nil
	case "USER":
		return LOG_USER, nil
	case "MAIL":
		return LOG_MAIL, nil
	case "DAEMON":
		return LOG_DAEMON, nil
	case "AUTH":
		return LOG_AUTH, nil
	case "SYSLOG":
		return LOG_SYSLOG, nil
	case "LPR":
		return LOG_LPR, nil
	case "NEWS":
		return LOG_NEWS, nil
	case "UUCP":
		return LOG_UUCP, nil
	case "CRON":
		return LOG_CRON, nil
	case "AUTHPRIV":
		return LOG_AUTHPRIV, nil
	case "FTP":
		return LOG_FTP, nil
	case "LOCAL0":
		return LOG_LOCAL0, nil
	case "LOCAL1":
		return LOG_LOCAL1, nil
	case "LOCAL2":
		return LOG_LOCAL2, nil
	case "LOCAL3":
		return LOG_LOCAL3, nil
	case "LOCAL4":
		return LOG_LOCAL4, nil
	case "LOCAL5":
		return LOG_LOCAL5, nil
	case "LOCAL6":
		return LOG_LOCAL6, nil
	case "LOCAL7":
		return LOG_LOCAL7, nil
	default:
		return 0, fmt.Errorf("invalid syslog facility: %s", facility)
	}
}

func parsePriority(priority string) (Priority, error) {
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

type Options struct {
	Connection string `short:"c" long:"network" description:"Connect to this network" choice:"tcp" choice:"udp" default:"udp"`
	Address    string `short:"n" long:"address" description:"Write to this remote syslog server" default:":514"`
	Priority   string `short:"p" long:"priority" description:"Mark given message with this priority" default:"user.notice"`
	Tag        string `short:"t" long:"tag" description:"Mark every line with this tag (default: $0)"`
	Hostname   string `short:"l" long:"hostname" description:"Override syslog sender with this name (default: hostname)"`
}

func main() {
	var opts Options
	args, err := flags.Parse(&opts)
	if err != nil {
		if fe, ok := err.(*flags.Error); ok && fe.Type == flags.ErrHelp {
			os.Exit(0)
		}
		log.Print(err)
		os.Exit(1)
	}

	priority, err := parsePriority(opts.Priority)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	w, err := Dial(opts.Connection, opts.Address, priority, opts.Tag, opts.Hostname)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	defer w.Close()

	message := strings.Join(args, " ")
	if len(message) > 0 {
		w.Write([]byte(message))
	}
}
