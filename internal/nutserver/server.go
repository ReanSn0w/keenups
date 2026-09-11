package nutserver

import (
	"bufio"
	"context"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/reansnow/keenups/internal/state"
)

type Server struct {
	Address     string
	UPSName     string
	Description string
	Username    string
	Password    string
	Store       *state.Store
	Logger      *slog.Logger
	Version     string
	attached    atomic.Int64
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.Address)
	if err != nil {
		return err
	}
	defer listener.Close()
	s.Logger.Info("NUT server listening", "address", listener.Addr(), "ups", s.UPSName)
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		go s.serveConn(conn)
	}
}

type session struct {
	username string
	password string
	attached bool
}

func (s *Server) serveConn(conn net.Conn) {
	defer conn.Close()
	sess := &session{}
	defer func() {
		if sess.attached {
			s.attached.Add(-1)
		}
	}()

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 1024), 64*1024)
	writer := bufio.NewWriter(conn)
	for scanner.Scan() {
		lines, closeConn := s.handle(sess, scanner.Text())
		for _, line := range lines {
			_, _ = writer.WriteString(line + "\n")
		}
		if err := writer.Flush(); err != nil || closeConn {
			return
		}
	}
}

func (s *Server) handle(sess *session, line string) ([]string, bool) {
	t, err := splitLine(line)
	if err != nil || len(t) == 0 {
		return []string{"ERR INVALID-ARGUMENT"}, false
	}
	cmd := strings.ToUpper(t[0])
	switch cmd {
	case "VER":
		return []string{fmt.Sprintf("Network UPS Tools upsd %s - https://networkupstools.org/", s.Version)}, false
	case "NETVER", "PROTVER":
		return []string{"1.3"}, false
	case "HELP":
		return []string{"Commands: HELP VER NETVER PROTVER GET LIST USERNAME PASSWORD LOGIN LOGOUT ATTACH DETACH"}, false
	case "USERNAME":
		if len(t) != 2 {
			return []string{"ERR INVALID-ARGUMENT"}, false
		}
		sess.username = t[1]
		return []string{"OK"}, false
	case "PASSWORD":
		if len(t) != 2 {
			return []string{"ERR INVALID-ARGUMENT"}, false
		}
		sess.password = t[1]
		return []string{"OK"}, false
	case "LOGIN", "ATTACH":
		if len(t) != 2 || t[1] != s.UPSName {
			return []string{"ERR UNKNOWN-UPS"}, false
		}
		if !s.authenticated(sess) {
			return []string{"ERR ACCESS-DENIED"}, false
		}
		if !sess.attached {
			sess.attached = true
			s.attached.Add(1)
		}
		return []string{"OK"}, false
	case "LOGOUT", "DETACH":
		if sess.attached {
			sess.attached = false
			s.attached.Add(-1)
		}
		return []string{"OK Goodbye"}, true
	case "MASTER", "PRIMARY", "FSD", "SET", "INSTCMD":
		return []string{"ERR ACCESS-DENIED"}, false
	case "STARTTLS":
		return []string{"ERR FEATURE-NOT-SUPPORTED"}, false
	case "GET":
		return s.handleGet(t), false
	case "LIST":
		return s.handleList(t), false
	default:
		return []string{"ERR UNKNOWN-COMMAND"}, false
	}
}

func (s *Server) authenticated(sess *session) bool {
	return secureEqual(sess.username, s.Username) && secureEqual(sess.password, s.Password)
}

func secureEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func (s *Server) handleGet(t []string) []string {
	if len(t) < 3 {
		return []string{"ERR INVALID-ARGUMENT"}
	}
	sub := strings.ToUpper(t[1])
	if t[2] != s.UPSName {
		return []string{"ERR UNKNOWN-UPS"}
	}
	snap := s.Store.Snapshot()
	switch sub {
	case "UPSDESC":
		return []string{fmt.Sprintf("UPSDESC %s %s", s.UPSName, quote(s.Description))}
	case "NUMLOGINS", "NUMATTACH":
		return []string{fmt.Sprintf("%s %s %d", sub, s.UPSName, s.attached.Load())}
	case "VAR":
		if len(t) != 4 {
			return []string{"ERR INVALID-ARGUMENT"}
		}
		if !snap.Ready {
			return []string{"ERR DATA-STALE"}
		}
		value, ok := snap.Variables[t[3]]
		if !ok {
			return []string{"ERR VAR-NOT-SUPPORTED"}
		}
		return []string{fmt.Sprintf("VAR %s %s %s", s.UPSName, t[3], quote(value))}
	case "TYPE":
		if len(t) != 4 {
			return []string{"ERR INVALID-ARGUMENT"}
		}
		value, ok := snap.Variables[t[3]]
		if !ok {
			return []string{"ERR VAR-NOT-SUPPORTED"}
		}
		if _, err := strconv.ParseFloat(value, 64); err == nil {
			return []string{fmt.Sprintf("TYPE %s %s NUMBER", s.UPSName, t[3])}
		}
		return []string{fmt.Sprintf("TYPE %s %s STRING:%d", s.UPSName, t[3], max(64, len(value)))}
	case "DESC":
		if len(t) != 4 {
			return []string{"ERR INVALID-ARGUMENT"}
		}
		if _, ok := snap.Variables[t[3]]; !ok {
			return []string{"ERR VAR-NOT-SUPPORTED"}
		}
		return []string{fmt.Sprintf("DESC %s %s %s", s.UPSName, t[3], quote("Unavailable"))}
	default:
		return []string{"ERR UNKNOWN-COMMAND"}
	}
}

func (s *Server) handleList(t []string) []string {
	if len(t) < 2 {
		return []string{"ERR INVALID-ARGUMENT"}
	}
	sub := strings.ToUpper(t[1])
	if sub == "UPS" {
		return []string{
			"BEGIN LIST UPS",
			fmt.Sprintf("UPS %s %s", s.UPSName, quote(s.Description)),
			"END LIST UPS",
		}
	}
	if len(t) != 3 || t[2] != s.UPSName {
		return []string{"ERR UNKNOWN-UPS"}
	}
	snap := s.Store.Snapshot()
	switch sub {
	case "VAR":
		if !snap.Ready {
			return []string{"ERR DATA-STALE"}
		}
		keys := make([]string, 0, len(snap.Variables))
		for key := range snap.Variables {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		lines := []string{fmt.Sprintf("BEGIN LIST VAR %s", s.UPSName)}
		for _, key := range keys {
			lines = append(lines, fmt.Sprintf("VAR %s %s %s", s.UPSName, key, quote(snap.Variables[key])))
		}
		return append(lines, fmt.Sprintf("END LIST VAR %s", s.UPSName))
	case "RW", "CMD", "CLIENT":
		return []string{
			fmt.Sprintf("BEGIN LIST %s %s", sub, s.UPSName),
			fmt.Sprintf("END LIST %s %s", sub, s.UPSName),
		}
	default:
		return []string{"ERR UNKNOWN-COMMAND"}
	}
}
