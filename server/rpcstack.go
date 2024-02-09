package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/cometbft/cometbft/libs/log"
)

// Config represents a small collection of configuration values to fine tune the
// P2P network layer of a protocol stack. These values can be further extended by
// all registered services.
type Config struct {
	Name string
	// HTTPHost is the host interface on which to start the HTTP RPC server. If this
	// field is empty, no HTTP API endpoint will be started.
	HTTPHost string

	// HTTPPort is the TCP port number on which to start the HTTP RPC server. The
	// default zero value is/ valid and will pick a port number randomly (useful
	// for ephemeral nodes).
	HTTPPort int `toml:",omitempty"`

	// HTTPPathPrefix specifies a path prefix on which http-rpc is to be served.
	HTTPPathPrefix string `toml:",omitempty"`

	// WSPathPrefix specifies a path prefix on which ws-rpc is to be served.
	WSPathPrefix string `toml:",omitempty"`

	// BatchRequestLimit is the maximum number of requests in a batch.
	BatchRequestLimit int `toml:",omitempty"`

	// BatchResponseMaxSize is the maximum number of bytes returned from a batched rpc call.
	BatchResponseMaxSize int `toml:",omitempty"`
}

type httpServer struct {
	log      log.Logger
	timeouts rpc.HTTPTimeouts
	mux      http.ServeMux // registered handlers go here

	mu       sync.Mutex
	server   *http.Server
	listener net.Listener // non-nil when server is running

	httpConfig *Config

	// HTTP RPC handler things.
	httpHandler http.Handler
	wsHandler   http.Handler

	// These are set by setListenAddr.
	endpoint string
	host     string
	port     int
}

const (
	shutdownTimeout = 5 * time.Second
)

func newHTTPServer(logger log.Logger, timeouts rpc.HTTPTimeouts) *httpServer {
	return &httpServer{log: logger, timeouts: timeouts}
}

// ListenAddr returns the listening address of the server.
func (h *httpServer) ListenAddr() string {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.listener != nil {
		return h.listener.Addr().String()
	}
	return h.endpoint
}

// setListenAddr configures the listening address of the server.
// The address can only be set while the server isn't running.
func (h *httpServer) setListenAddr(host string, port int) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.listener != nil && (host != h.host || port != h.port) {
		return fmt.Errorf("HTTP server already running on %s", h.endpoint)
	}

	h.host, h.port = host, port
	h.endpoint = net.JoinHostPort(host, fmt.Sprintf("%d", port))
	return nil
}

// start starts the HTTP server if it is enabled and not already running.
func (h *httpServer) start(c *Config, apis []rpc.API) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.endpoint == "" || h.listener != nil {
		return nil // already running or not configured
	}

	// Initialize the server.
	//nolint // TODO: add a proper timeout for the server
	h.server = &http.Server{Handler: h}
	h.server = &http.Server{
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second, // Example: 5 seconds timeout
		// Other configurations...
	}

	if h.timeouts != (rpc.HTTPTimeouts{}) {
		h.server.ReadTimeout = h.timeouts.ReadTimeout
		h.server.ReadHeaderTimeout = h.timeouts.ReadHeaderTimeout
		h.server.WriteTimeout = h.timeouts.WriteTimeout
		h.server.IdleTimeout = h.timeouts.IdleTimeout
	}

	// Create RPC server and handler.
	srv := rpc.NewServer()
	srv.SetBatchLimits(c.BatchRequestLimit, c.BatchResponseMaxSize)
	if err := RegisterApis(apis, srv); err != nil {
		return err
	}
	h.wsHandler = srv.WebsocketHandler([]string{})
	h.httpHandler = srv

	listener, err := net.Listen("tcp", h.endpoint)
	if err != nil {
		return err
	}

	h.listener = listener
	h.httpConfig = c

	// Start the server.
	go func() {
		if err := h.server.Serve(h.listener); err != nil && err != http.ErrServerClosed {
			h.log.Error("HTTP server failed to start", "error", err)
		}
	}()

	h.log.Info("Execution engine rpc server enabled",
		"http", fmt.Sprintf("http://%v", listener.Addr()),
		"ws", fmt.Sprintf("ws://%v%s", listener.Addr(), c.WSPathPrefix))

	return nil
}

func (h *httpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// check if ws request and serve if ws enabled
	ws := h.wsHandler
	if ws != nil && isWebsocket(r) {
		if checkPath(r, h.httpConfig.WSPathPrefix) {
			ws.ServeHTTP(w, r)
		}
		return
	}

	// if http-rpc is enabled, try to serve request
	rpcHandler := h.httpHandler
	if rpcHandler != nil {
		// First try to route in the mux.
		// Requests to a path below root are handled by the mux,
		// which has all the handlers registered via Node.RegisterHandler.
		// These are made available when RPC is enabled.
		muxHandler, pattern := h.mux.Handler(r)
		if pattern != "" {
			muxHandler.ServeHTTP(w, r)
			return
		}

		if checkPath(r, h.httpConfig.HTTPPathPrefix) {
			rpcHandler.ServeHTTP(w, r)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

// checkPath checks whether a given request URL matches a given path prefix.
func checkPath(r *http.Request, path string) bool {
	// if no prefix has been specified, request URL must be on root
	if path == "" {
		return r.URL.Path == "/"
	}
	// otherwise, check to make sure prefix matches
	return len(r.URL.Path) >= len(path) && r.URL.Path[:len(path)] == path
}

// stop shuts down the HTTP server.
func (h *httpServer) stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.listener == nil {
		return // not running
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	err := h.server.Shutdown(ctx)
	if err != nil && err == ctx.Err() {
		h.log.Info("HTTP server graceful shutdown timed out")
		if err := h.server.Close(); err != nil {
			h.log.Error("HTTP server forced to shutdown", "error", err)
		}
	}

	if err := h.listener.Close(); err != nil {
		h.log.Error("Failed to close listener", "error", err)
	}
	h.log.Info("HTTP server stopped", "endpoint", h.listener.Addr())

	// Clear out everything to allow re-configuring it later.
	h.host, h.port, h.endpoint = "", 0, ""
	h.server, h.listener = nil, nil
}

// isWebsocket checks the header of an http request for a websocket upgrade request.
func isWebsocket(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket") &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

// RegisterApis checks the given modules' availability, generates an allowlist based on the allowed modules,
// and then registers all of the APIs exposed by the services.
func RegisterApis(apis []rpc.API, srv *rpc.Server) error {
	// Register all the APIs exposed by the services
	for _, api := range apis {
		if err := srv.RegisterName(api.Namespace, api.Service); err != nil {
			return err
		}
	}
	return nil
}
