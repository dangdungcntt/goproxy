package goproxy

import (
	"github.com/go-chi/chi/v5"
	"github.com/rs/cors"
	"net/http/httputil"
	"net/url"
)

type RunOption func(server *Server)

func WithTargetURL(targetURL *url.URL) RunOption {
	return func(s *Server) {
		s.targetURL = targetURL
	}
}

func WithRewrite(rewrite RewriteFunc) RunOption {
	return func(s *Server) {
		s.rewrite = rewrite
	}
}

func WithReverseProxy(reverseProxy *httputil.ReverseProxy) RunOption {
	return func(s *Server) {
		s.reverseProxy = reverseProxy
	}
}

func WithPort(port int) RunOption {
	return func(s *Server) {
		s.port = port
	}
}

func WithRouter(router func(mux *chi.Mux)) RunOption {
	return func(s *Server) {
		router(s.mux)
	}
}

func WithCors(corsOpt cors.Options) RunOption {
	return WithRouter(func(mux *chi.Mux) {
		mux.Use(cors.New(corsOpt).Handler)
	})
}
