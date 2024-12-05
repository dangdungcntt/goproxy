package goproxy

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type RewriteFunc func(*httputil.ProxyRequest)

type Server struct {
	mux          *chi.Mux
	port         int
	targetURL    *url.URL
	rewrite      RewriteFunc
	reverseProxy *httputil.ReverseProxy
}

func NewServer(opt ...Option) *Server {
	s := &Server{
		mux:  chi.NewRouter(),
		port: 3000,
	}

	s.mux.Use(middleware.Logger)
	s.mux.Use(middleware.Recoverer)
	s.mux.Use(middleware.RequestID)

	for _, o := range opt {
		o(s)
	}

	switch {
	case s.targetURL != nil:
		s.reverseProxy = &httputil.ReverseProxy{
			Rewrite: func(r *httputil.ProxyRequest) {
				r.SetURL(s.targetURL)
				if !strings.HasSuffix(s.targetURL.RawPath, "/") {
					r.Out.URL.Path = strings.TrimRight(r.Out.URL.Path, "/")
					r.Out.URL.RawPath = strings.TrimRight(r.Out.URL.RawPath, "/")
				}
			},
		}
	case s.rewrite != nil:
		s.reverseProxy = &httputil.ReverseProxy{Rewrite: s.rewrite}
	default:
		if s.reverseProxy == nil {
			log.Fatal("missing configuration for reverse proxy")
		}
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
