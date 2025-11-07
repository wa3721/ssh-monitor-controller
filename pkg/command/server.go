package command

import (
	"context"
	"io"

	"k8s.io/apiserver/pkg/server/healthz"
	ctrl "sigs.k8s.io/controller-runtime"

	"net/http"
	"strconv"
	"time"
)

var DefaultPort = 8082
var DefaultCommandChanLength = 1000
var commandServerLog = ctrl.Log.WithName("command-server")

var CmdChannel <-chan *string

type Server interface {
	Start(ctx context.Context) error
	registerHandlers()
	GetCommand() <-chan *string
}

type DefaultServer struct {
	Host     string
	Port     int
	ServeMux *http.ServeMux
	Command  chan *string
}

type Options func(server *DefaultServer)

func WithHost(host string) func(server *DefaultServer) {
	return func(server *DefaultServer) {
		server.Host = host
	}
}

func WithPort(port int) Options {
	return func(server *DefaultServer) {
		server.Port = port
	}
}

func WithServeMux(mux *http.ServeMux) Options {
	return func(server *DefaultServer) {
		server.ServeMux = mux
	}
}
func WithCommandChanLength(length int) Options {
	return func(server *DefaultServer) {
		server.Command = make(chan *string, length)
	}
}

func NewDefaultServer(options ...Options) Server {
	server := &DefaultServer{
		Port:     DefaultPort,
		Command:  make(chan *string, DefaultCommandChanLength),
		ServeMux: http.NewServeMux(),
	}
	for _, option := range options {
		option(server)
	}
	return server
}

func (s *DefaultServer) registerHandlers() {
	livezChecks := []healthz.HealthChecker{healthz.PingHealthz}
	healthz.InstallPathHandler(s.ServeMux, "/livez", livezChecks...)

	readyzChecks := []healthz.HealthChecker{healthz.PingHealthz}
	healthz.InstallPathHandler(s.ServeMux, "/readyz", readyzChecks...)

	// 命令接收路由
	s.ServeMux.HandleFunc("/", s.commandHandler)
}

// 添加缺失的处理函数实现
func (s *DefaultServer) commandHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		commandServerLog.Error(err, "Failed to read body")
	}
	defer r.Body.Close()
	s.Command <- func() *string {
		b := string(body)
		return &b
	}()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok", "message": "Command server is running"}`))
}

func (s *DefaultServer) Start(ctx context.Context) error {

	commandServerLog.Info("Starting command server")
	s.registerHandlers()
	srv := newHttpServer(s.ServeMux)
	idleConnsClosed := make(chan struct{})
	go func() {
		<-ctx.Done()
		commandServerLog.Info("Shutting down command server with timeout of 1 minute")

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			// Error from closing listeners, or context timeout
			commandServerLog.Error(err, "error shutting down the HTTP server")
		}
		close(idleConnsClosed)
	}()
	CmdChannel = s.Command
	srv.Addr = ":" + strconv.Itoa(s.Port)
	if err := srv.ListenAndServe(); err != nil {
		return err
	}
	commandServerLog.Info("Started command server")
	<-idleConnsClosed
	return nil
}

func (s *DefaultServer) GetCommand() <-chan *string {
	return s.Command
}

func newHttpServer(handler http.Handler) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       90 * time.Second,
	}
}
