package editormcp

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func (s *Server) Handler(token, authority string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.URL.Path != "/mcp" {
			http.NotFound(w, r)
			return
		}
		if r.Host != authority {
			http.Error(w, "Invalid Host", http.StatusForbidden)
			return
		}
		if origins := r.Header.Values("Origin"); len(origins) > 0 {
			u, err := url.Parse(origins[0])
			if len(origins) != 1 || err != nil || u.Scheme != "http" || u.Host != authority || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
				http.Error(w, "Invalid Origin", http.StatusForbidden)
				return
			}
		}
		if token == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+token)) != 1 {
			http.Error(w, "Bearer token required", http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if ct := strings.Split(r.Header.Get("Content-Type"), ";")[0]; ct != "application/json" {
			http.Error(w, "Expected application/json", http.StatusUnsupportedMediaType)
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxMessage))
		if err != nil {
			http.Error(w, "Request too large", http.StatusRequestEntityTooLarge)
			return
		}
		var message struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params struct {
				Name string                     `json:"name"`
				Meta map[string]json.RawMessage `json:"_meta"`
			} `json:"params"`
		}
		if json.Unmarshal(body, &message) == nil && len(message.ID) > 0 {
			var declared string
			_ = json.Unmarshal(message.Params.Meta["io.modelcontextprotocol/protocolVersion"], &declared)
			if declared == ProtocolVersion || r.Header.Get("MCP-Protocol-Version") == ProtocolVersion {
				code, reason := 0, ""
				if r.Header.Get("MCP-Protocol-Version") == "" || r.Header.Get("Mcp-Method") == "" {
					code, reason = -32602, "Missing request metadata headers"
				} else if r.Header.Get("Mcp-Method") != message.Method || r.Header.Get("MCP-Protocol-Version") != declared || message.Method == "tools/call" && r.Header.Get("Mcp-Name") != message.Params.Name {
					code, reason = -32020, "Request headers disagree with JSON-RPC body"
				}
				if code != 0 {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					_ = json.NewEncoder(w).Encode(response{JSONRPC: "2.0", ID: message.ID, Error: &rpcError{Code: code, Message: reason}})
					return
				}
			}
		}
		b, status := s.Handle(r.Context(), body, r.Header.Get("MCP-Protocol-Version"))
		if b != nil {
			w.Header().Set("Content-Type", "application/json")
		}
		w.WriteHeader(status)
		_, _ = w.Write(b)
	})
}

// RecordName is the one file in the profile that keeps the MCP bearer token
// and the last port across editor starts.
const RecordName = "mapedit-mcp.json"

// Listener is a running loopback endpoint. Notes lists what a start decided
// that the player should know: a new or replaced record, or a port that was
// taken.
type Listener struct {
	URL, Token, TokenFile string
	Notes                 []string
	server                *http.Server
	done                  chan error
}

// record is the stored form of RecordName. URL repeats the last address so a
// client reads one file; Port is what the next start tries first.
type record struct {
	URL   string `json:"url"`
	Token string `json:"token"`
	Port  int    `json:"port"`
}

// readRecord returns the stored record, or why it cannot be used.
func readRecord(file string) (record, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return record{}, err
	}
	var r record
	if err := json.Unmarshal(b, &r); err != nil {
		return record{}, fmt.Errorf("malformed: %v", err)
	}
	if raw, err := hex.DecodeString(r.Token); err != nil || len(raw) != 32 {
		return record{}, fmt.Errorf("malformed: token is not 64 hex digits")
	}
	if r.Port < 0 || r.Port > 65535 {
		return record{}, fmt.Errorf("malformed: port %d", r.Port)
	}
	return r, nil
}

// writeRecord replaces file with r: a temporary file in the same folder that
// only the user can read, then a rename, so a crash leaves the old record or
// the new one.
func writeRecord(file string, r record) error {
	content, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if old, err := os.ReadFile(file); err == nil && string(old) == string(content)+"\n" {
		return nil
	}
	f, err := os.CreateTemp(filepath.Dir(file), ".mapedit-mcp-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, werr := f.Write(append(content, '\n'))
	if werr == nil {
		werr = f.Sync()
	}
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = os.Rename(tmp, file)
	}
	if werr != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("write MCP record: %v", werr)
	}
	return nil
}

// Listen opens the loopback endpoint and records its token and port in
// profile. The token comes from the record of an earlier start unless newToken
// asks for a new one or the record cannot be used. An address with port 0 first
// tries the recorded port and takes a free one when that port is busy.
func (s *Server) Listen(address, profile string, newToken bool) (*Listener, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil || host != "127.0.0.1" {
		return nil, fmt.Errorf("MCP address must be 127.0.0.1:port")
	}
	if err := os.MkdirAll(profile, 0700); err != nil {
		return nil, err
	}
	file := filepath.Clean(filepath.Join(profile, RecordName))
	var notes []string
	stored, readErr := readRecord(file)
	switch {
	case readErr == nil && !newToken:
	case newToken:
		notes = append(notes, "new bearer token recorded (-mcp-new-token)")
	case errors.Is(readErr, os.ErrNotExist):
		notes = append(notes, "no earlier MCP record; new bearer token recorded")
	default:
		notes = append(notes, fmt.Sprintf("MCP record %s unusable (%v); replaced with a new bearer token", file, readErr))
	}
	token := stored.Token
	if readErr != nil || newToken {
		secret := make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			return nil, err
		}
		token = hex.EncodeToString(secret)
	}
	var l net.Listener
	if port == "0" && readErr == nil && stored.Port > 0 {
		l, err = net.Listen("tcp4", net.JoinHostPort(host, strconv.Itoa(stored.Port)))
		if err != nil {
			l = nil
			notes = append(notes, fmt.Sprintf("recorded port %d is taken; using a free port", stored.Port))
		}
	}
	if l == nil {
		if l, err = net.Listen("tcp4", address); err != nil {
			return nil, err
		}
	}
	authority := l.Addr().String()
	endpoint := "http://" + authority + "/mcp"
	if err := writeRecord(file, record{URL: endpoint, Token: token, Port: l.Addr().(*net.TCPAddr).Port}); err != nil {
		_ = l.Close()
		return nil, err
	}
	server := &http.Server{Handler: s.Handler(token, authority), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 8192}
	result := &Listener{URL: endpoint, Token: token, TokenFile: file, Notes: notes, server: server, done: make(chan error, 1)}
	go func() { result.done <- server.Serve(l) }()
	return result, nil
}

func (l *Listener) Close() error {
	err := l.server.Close()
	<-l.done
	return err
}
