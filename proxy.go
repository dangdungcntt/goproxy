package goproxy

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
	"net/http/httputil"
)

type RewriteFunc func(*httputil.ProxyRequest)

type Server struct {
	mux          *chi.Mux
	port         int
	reverseProxy *httputil.ReverseProxy
}

func NewServer(rewrite RewriteFunc, opt ...RunOption) *Server {
	s := &Server{
		mux:  chi.NewRouter(),
		port: 3000,
		reverseProxy: &httputil.ReverseProxy{
			Rewrite: rewrite,
		},
	}

	s.mux.Use(middleware.Recoverer)
	s.mux.Use(middleware.RequestID)
	s.mux.Use(middleware.Logger)

	for _, o := range opt {
		o(s)
	}

	s.mux.HandleFunc("/", func(writer http.ResponseWriter, r *http.Request) {
		s.reverseProxy.ServeHTTP(writer, r)
	})

	return s
}

func (s *Server) Run() error {
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), s)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
