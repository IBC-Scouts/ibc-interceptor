package rpcee

import (
	"net"
	"strconv"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/libs/service"
)

type EERPCServer struct {
	service.BaseService

	config  *Config
	log     log.Logger
	rpcAPIs []rpc.API
	srv     *httpServer
}

func DefaultConfig(addr string) *Config {
	host, portstr, err := net.SplitHostPort(addr)
	if err != nil {
		panic(err)
	}

	port, err := strconv.ParseInt(portstr, 10, 32)
	if err != nil {
		panic(err)
	}

	c := Config{
		Name:                 "Execution-Engine",
		HTTPHost:             host,
		HTTPPort:             int(port),
		HTTPPathPrefix:       "/",
		WSPathPrefix:         "/websocket",
		BatchRequestLimit:    4,
		BatchResponseMaxSize: 1000000,
	}
	return &c
}

// New creates a new server
func NewEeRPCServer(conf *Config, apis []rpc.API, logger log.Logger) *EERPCServer {
	s := &EERPCServer{
		config:  conf,
		log:     logger,
		rpcAPIs: apis,
		srv:     newHTTPServer(logger, rpc.DefaultHTTPTimeouts),
	}

	baseService := *service.NewBaseService(logger, s.ServiceName(), s)
	s.BaseService = baseService
	return s
}

func (s *EERPCServer) OnStart() error {
	if err := s.srv.setListenAddr(s.config.HTTPHost, s.config.HTTPPort); err != nil {
		return err
	}
	if err := s.srv.start(s.config, s.rpcAPIs); err != nil {
		s.log.Error("failed to serve RPC server", "err", err)
		return err
	}
	return nil
}

func (s *EERPCServer) OnStop() {
	s.srv.stop()
}

// Address returns the address the server is listening on
//
// Must be called after server.Start()
func (s *EERPCServer) Address() net.Addr {
	if s.srv == nil || s.srv.listener == nil {
		return nil
	}
	return s.srv.listener.Addr()
}

func (s *EERPCServer) ServiceName() string {
	return s.config.Name
}
