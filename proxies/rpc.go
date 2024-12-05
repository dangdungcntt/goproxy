package proxies

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/dangdungcntt/goproxy"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
)

type ClientIPResolver func(r *http.Request) string

var DefaultClientIPResolver = func(r *http.Request) string {
	return r.RemoteAddr
}

func MultiChainEthereumRPC(rpcMap map[int]string, useQueryParams ...string) (goproxy.RunOption, goproxy.RunOption) {
	rpcURLMap := make(map[int]*url.URL)

	for chainID, rpcURL := range rpcMap {
		u, err := url.Parse(rpcURL)
		if err != nil {
			log.Fatalf("invalid rpc url: %s", rpcURL)
		}
		u.RawPath = u.Path
		rpcURLMap[chainID] = u
	}

	chainIDResolver := func(r *httputil.ProxyRequest, rpcReq *rpcRequest) (int, error) {
		return rpcReq.ChainID, nil
	}
	if len(useQueryParams) > 0 && useQueryParams[0] != "" {
		chainIDResolver = func(r *httputil.ProxyRequest, _ *rpcRequest) (int, error) {
			chainID, _ := strconv.ParseInt(r.In.URL.Query().Get(useQueryParams[0]), 10, 64)
			return int(chainID), nil
		}
	}

	return goproxy.WithRewrite(func(r *httputil.ProxyRequest) {
			if r.In.Method != http.MethodPost {
				redirectToRPCError(r, 0, "method not allowed")
				return
			}

			rpcReq := &rpcRequest{}
			if err := render.Bind(r.In, rpcReq); err != nil {
				redirectToRPCError(r, rpcReq.ID, err.Error())
				return
			}

			chainID, err := chainIDResolver(r, rpcReq)
			if err != nil {
				redirectToRPCError(r, rpcReq.ID, err.Error())
				return
			}

			u, ok := rpcURLMap[chainID]
			if !ok {
				redirectToRPCError(r, rpcReq.ID, fmt.Sprintf("Unsupported chain ID: %d", chainID))
				return
			}

			buf := &bytes.Buffer{}
			enc := json.NewEncoder(buf)
			enc.SetEscapeHTML(true)

			rpcReq.ChainID = 0
			_ = enc.Encode(rpcReq)

			r.Out.Body = io.NopCloser(buf)
			r.Out.ContentLength = int64(buf.Len())
			r.Out.Header.Set("Content-Length", fmt.Sprintf("%d", r.Out.ContentLength))

			setXForwardedFor(r)
			r.SetURL(u)
			r.Out.Header.Set("X-Forwarded-Host", u.Host)
			if !strings.HasSuffix(u.RawPath, "/") {
				r.Out.URL.Path = strings.TrimRight(r.Out.URL.Path, "/")
				r.Out.URL.RawPath = strings.TrimRight(r.Out.URL.RawPath, "/")
			}
		}), goproxy.WithRouter(func(mux *chi.Mux) {
			mux.Get("/__rpc_error", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(200)
				_, _ = w.Write([]byte(strings.TrimSpace(r.URL.Query().Get("response"))))
			})
		})
}

type rpcRequest struct {
	ID      int    `json:"id"`
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
	ChainID int    `json:"chainId,omitempty"`
}

func (r2 *rpcRequest) Bind(_ *http.Request) error {
	return nil
}

type rpcResponse[T any] struct {
	ID      int       `json:"id"`
	JSONRPC string    `json:"jsonrpc"`
	Result  T         `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func redirectToRPCError(r *httputil.ProxyRequest, id int, message string) {
	r.Out.URL.Scheme = "http"
	r.Out.URL.Host = r.In.Host
	r.Out.URL.Path = strings.TrimRight(r.In.URL.Path, "/") + "/__rpc_error"
	r.Out.URL.RawPath = strings.TrimRight(r.In.URL.RawPath, "/") + "/__rpc_error"
	r.Out.Method = http.MethodGet

	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(true)

	_ = enc.Encode(rpcResponse[*string]{
		ID:      id,
		JSONRPC: "2.0",
		Error: &rpcError{
			Code:    -32000,
			Message: message,
		},
	})

	q := r.Out.URL.Query()
	q.Set("response", buf.String())
	r.Out.URL.RawQuery = q.Encode()
	r.Out.ContentLength = 0
	r.Out.Header.Del("Content-Length")
}

func setXForwardedFor(r *httputil.ProxyRequest) {
	remoteAddr := DefaultClientIPResolver(r.In)

	clientIP, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		prior := r.In.Header["X-Forwarded-For"]
		if len(prior) > 0 {
			clientIP = strings.Join(prior, ", ") + ", " + clientIP
		}
		r.Out.Header.Set("X-Forwarded-For", clientIP)
	} else {
		r.Out.Header.Del("X-Forwarded-For")
	}
	if r.In.TLS == nil {
		r.Out.Header.Set("X-Forwarded-Proto", "http")
	} else {
		r.Out.Header.Set("X-Forwarded-Proto", "https")
	}
}
