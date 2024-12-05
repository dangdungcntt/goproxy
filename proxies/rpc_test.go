package proxies_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/dangdungcntt/goproxy"
	"github.com/dangdungcntt/goproxy/proxies"
	"github.com/go-chi/render"
	"github.com/stretchr/testify/assert"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
)

type rpcRequest struct {
	ID      int    `json:"id"`
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
	ChainID int    `json:"chainId,omitempty"`
}

type rpcResponse[T any] struct {
	ID      int    `json:"id"`
	JSONRPC string `json:"jsonrpc"`
	Result  T      `json:"result"`
}

func TestMultiChainEthereumRPC(t *testing.T) {
	rpcMap := map[int]string{
		//43114: "https://api.avax.network/ext/bc/C/rpc",
		//1:     "https://rpc.ankr.com/eth",
		//43113: "https://api.avax-test.network/ext/bc/C/rpc",
		42161: "https://arbitrum.llamarpc.com",
	}

	rpcServer := httptest.NewServer(goproxy.NewServer(
		proxies.MultiChainEthereumRPC(rpcMap),
	))
	defer rpcServer.Close()

	rpcServerClient := rpcServer.Client()

	for chainID, _ := range rpcMap {
		body := &bytes.Buffer{}
		enc := json.NewEncoder(body)
		enc.SetEscapeHTML(true)

		_ = enc.Encode(rpcRequest{
			ID:      0,
			JSONRPC: "2.0",
			Method:  "eth_chainId",
			ChainID: chainID,
		})

		req, err := http.NewRequest("POST", rpcServer.URL, body)
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := rpcServerClient.Do(req)
		assert.NoError(t, err)

		var res rpcResponse[string]

		err = render.DecodeJSON(resp.Body, &res)
		assert.NoError(t, err, fmt.Sprintf("%d", chainID))

		chainIDResult, ok := big.NewInt(0).SetString(res.Result, 0)
		assert.True(t, ok)
		assert.Equal(t, int64(chainID), chainIDResult.Int64())
	}

}
