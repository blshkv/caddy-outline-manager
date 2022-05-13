package outline

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/caddyserver/caddy/v2"
	caddycmd "github.com/caddyserver/caddy/v2/cmd"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"

	"go.uber.org/zap"

	"github.com/imgk/caddy-outline-manager/outline"
)

func init() {
	caddy.RegisterModule(Handler{})

	caddycmd.RegisterCommand(caddycmd.Command{
		Name:  "outline",
		Func:  cmdOutlineManager,
		Usage: "<command> server",
		Short: "Start Outline manager",
		Long:  "",
		Flags: func() *flag.FlagSet {
			fs := flag.NewFlagSet("outline", flag.ExitOnError)
			fs.String("server", "", "server url")
			fs.String("username", "", "username")
			fs.String("password", "", "password")
			return fs
		}(),
	})
}

var configTemplate = template.Must(template.New("").Parse(`{
    "http_port": 80,
    "https_port": 443,
    "servers": {
        "outline": {
            "listen": [":80"],
            "routes": [
                {
                    "handle": [
                        {
                            "handler": "outline_manager",
                            "servers": ["{{ .Servers }}"],
                            "username": "{{ .Username }}",
                            "password": "{{ .RawPass }}"
                        }
                    ]
                }
            ]
        }
    }
}
`))

func cmdOutlineManager(fl caddycmd.Flags) (int, error) {
	caddy.TrapSignals()

	user := fl.String("username")
	pass := fl.String("password")
	server := fl.String("server")

	type UserPass struct {
		Username string `json:"username"`
		RawPass  string `json:"rawpass"`
		Password string `json:"password"`
		Servers  string `json:"_,omitempty"`
	}
	userpass := UserPass{}
	if user == "" || pass == "" {
		b, err := ioutil.ReadFile("outline-manager.json")
		if err != nil {
			return caddy.ExitCodeFailedStartup, err
		}
		if err := json.Unmarshal(b, &userpass); err != nil {
			return caddy.ExitCodeFailedStartup, err
		}
	} else {
		userpass.Username = user
		userpass.RawPass = pass

		cmd := exec.Command(os.Args[0], "hash-password", "-plaintext", pass)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return caddy.ExitCodeFailedStartup, err
		}
		defer stdout.Close()

		sigCh := make(chan struct{})
		errCh := make(chan error)
		go func() {
			sigCh <- struct{}{}
			b, err := ioutil.ReadAll(stdout)
			if err != nil {
				errCh <- err
			}
			userpass.Password = string(b[:len(b)-1])
			errCh <- nil
		}()

		<-sigCh
		if err := cmd.Run(); err != nil {
			return caddy.ExitCodeFailedStartup, err
		}
		if err := <-errCh; err != nil {
			return caddy.ExitCodeFailedStartup, err
		}
	}

	buffer := bytes.NewBuffer(nil)
	userpass.Servers = server
	if err := configTemplate.Execute(buffer, userpass); err != nil {
		return caddy.ExitCodeFailedStartup, err
	}

	persist := false
	if err := caddy.Run(&caddy.Config{
		Admin: &caddy.AdminConfig{
			Disabled: true,
			Config: &caddy.ConfigSettings{
				Persist: &persist,
			},
		},
		AppsRaw: caddy.ModuleMap{
			"http": json.RawMessage(buffer.Bytes()),
		},
	}); err != nil {
		return caddy.ExitCodeFailedStartup, err
	}

	select {}
}

// Handler implements an HTTP handler that ...
type Handler struct {
	Servers  []string `json:"servers"`
	Username string   `json:"username,omitempty"`
	Password string   `json:"password,omitempty"`

	logger *zap.Logger
	server *outline.Server
}

// CaddyModule returns the Caddy module information.
func (Handler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.outline_manager",
		New: func() caddy.Module { return new(Handler) },
	}
}

// Provision implements caddy.Provisioner.
func (m *Handler) Provision(ctx caddy.Context) (err error) {
	m.logger = ctx.Logger(m)

	if len(m.Servers) == 0 {
		err = errors.New("no server for outline manager")
	}
	m.logger.Info(fmt.Sprintf("set up username: %v, password: %v", m.Username, m.Password))

	// Parse all server url
	servers := map[uint32]*outline.OutlineServer{}
	for _, url := range m.Servers {
		id := rand.Uint32()
		server := outline.NewOutlineServer(id, url, m.logger)
		if err := server.GetServerInfo(); err != nil {
			m.logger.Error(fmt.Sprintf("failed to get server info from server: %v, error: %v", url, err))
			continue
		}
		if err := server.GetAllUser(); err != nil {
			m.logger.Error(fmt.Sprintf("failed to get user from server: %v, error: %v", url, err))
			continue
		}
		servers[id] = server

		m.logger.Info("http://127.0.0.1:80/outline/manager")
		break
	}

	if len(servers) == 0 {
		err = errors.New("no available outline server")
		return
	}
	m.server = outline.NewServer(servers, m.logger)
	return
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (m *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	if strings.HasPrefix(r.URL.Path, "/login") {
		return m.Login(w, r)
	}
	if strings.HasPrefix(r.URL.Path, "/outline/manager/set/admin") {
		return m.ChangeUserPass(w, r)
	}
	if handler, ok := m.server.Handler(r); ok {
		handler.ServeHTTP(w, r)
		return nil
	}
	return next.ServeHTTP(w, r)
}

// Interface guards
var (
	_ caddyhttp.MiddlewareHandler = (*Handler)(nil)
)

func (m *Handler) Login(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case http.MethodGet:
		io.WriteString(w, login)
		return nil
	case http.MethodPost:
		user := r.URL.Query().Get("user")
		pass := r.URL.Query().Get("pass")
		if user != "" && pass != "" {
			m.logger.Info(fmt.Sprintf("try with %v: %v", user, pass))
			if user == m.Username && pass == m.Password {
				w.WriteHeader(http.StatusOK)
				return nil
			}
		}
	default:
	}
	http.HandlerFunc(http.NotFound).ServeHTTP(w, r)
	return nil
}

func (m *Handler) ChangeUserPass(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return nil
	}

	user := r.URL.Query().Get("user")
	pass := r.URL.Query().Get("pass")
	if user == "" || pass == "" {
		m.logger.Info("no user password for change")
		return nil
	}
	m.Username = user
	m.Password = pass

	m.logger.Info(fmt.Sprintf("change user pass to %v: %v", user, pass))

	type UserPass struct {
		Username string `json:"username"`
		RawPass  string `json:"rawpass"`
		Password string `json:"password"`
	}
	userpass := &UserPass{Username: user, RawPass: pass, Password: ""}

	cmd := exec.Command(os.Args[0], "hash-password", "-plaintext", pass)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer stdout.Close()

	sigCh := make(chan struct{})
	errCh := make(chan error)
	go func() {
		sigCh <- struct{}{}
		b, err := ioutil.ReadAll(stdout)
		if err != nil {
			errCh <- err
		}
		userpass.Password = string(b[:len(b)-1])
		errCh <- nil
	}()

	<-sigCh
	if err := cmd.Run(); err != nil {
		return err
	}
	if err := <-errCh; err != nil {
		return err
	}

	b, err := json.Marshal(&userpass)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile("outline-manager.json", b, 0644); err != nil {
		return err
	}

	return nil
}
