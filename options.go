package goproxy

import (
	"github.com/go-chi/chi/v5"
	"github.com/rs/cors"
	"net/http/httputil"
	"net/url"
)

type Option func(s *Server)

func WithTargetURL(t *url.URL) Option {
	return func(s *Server) {
		s.targetURL = t
	}
}

func WithRewrite(r RewriteFunc) Option {
	return func(s *Server) {
		s.rewrite = r
	}
}

func WithReverseProxy(proxy *httputil.ReverseProxy) Option {
	return func(s *Server) {
		s.reverseProxy = proxy
	}
}

func WithPort(p int) Option {
	return func(s *Server) {
		s.port = p
	}
}

func WithRouter(r func(mux *chi.Mux)) Option {
	return func(s *Server) {
		r(s.mux)
	}
}

func WithCors(o cors.Options) Option {
	return WithRouter(func(mux *chi.Mux) {
		mux.Use(cors.New(o).Handler)
	})
}
