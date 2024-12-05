package proxies

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/dangdungcntt/goproxy"
	"github.com/go-chi/render"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

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

func MultiChainEthereumRPC(rpcMap map[int]string) goproxy.RewriteFunc {
	rpcURLMap := make(map[int]*url.URL)

	for chainID, rpcURL := range rpcMap {
		u, err := url.Parse(rpcURL)
		if err != nil {
			log.Fatalf("invalid rpc url: %s", rpcURL)
		}
		u.RawPath = u.Path
		rpcURLMap[chainID] = u
	}

	return func(r *httputil.ProxyRequest) {
		if r.In.Method != http.MethodPost {
			return
		}

		body := &rpcRequest{}
		if err := render.Bind(r.In, body); err != nil {
			return
		}

		u, ok := rpcURLMap[body.ChainID]
		if !ok {
			return
		}
		body.ChainID = 0

		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(true)

		if err := enc.Encode(body); err != nil {
			return
		}

		r.Out.Body = io.NopCloser(buf)
		r.Out.ContentLength = int64(buf.Len())
		r.Out.Header.Set("Content-Length", fmt.Sprintf("%d", r.Out.ContentLength))

		r.SetXForwarded()
		r.SetURL(u)
		if !strings.HasSuffix(u.RawPath, "/") {
			r.Out.URL.Path = strings.TrimRight(r.Out.URL.Path, "/")
			r.Out.URL.RawPath = strings.TrimRight(r.Out.URL.RawPath, "/")
		}
	}
}
