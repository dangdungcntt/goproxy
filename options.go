package goproxy

import (
	"github.com/rs/cors"
)

type RunOption func(server *Server)

func WithCors(corsOpt cors.Options) RunOption {
	return func(s *Server) {
		s.mux.Use(cors.New(corsOpt).Handler)
	}
}

func WithPort(port int) RunOption {
	return func(s *Server) {
		s.port = port
	}
}
