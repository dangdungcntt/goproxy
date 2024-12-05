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
	"net/url"
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
	ID      int       `json:"id"`
	JSONRPC string    `json:"jsonrpc"`
	Result  T         `json:"result"`
	Error   *rpcError `json:"error"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func sendRequest[T any](client *http.Client, url string, method string, rpcReq rpcRequest) (*rpcResponse[T], error) {
	body := &bytes.Buffer{}
	enc := json.NewEncoder(body)
	enc.SetEscapeHTML(true)

	err := enc.Encode(rpcReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	var res rpcResponse[T]

	err = render.DecodeJSON(resp.Body, &res)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	return &res, nil
}

func getRpcMap() map[int]string {
	return map[int]string{
		43114: "https://api.avax.network/ext/bc/C/rpc",
		1:     "https://rpc.ankr.com/eth",
		43113: "https://api.avax-test.network/ext/bc/C/rpc",
		42161: "https://arbitrum.llamarpc.com",
	}
}

func TestMultiChainEthereumRPCValidRequests(t *testing.T) {
	rpcServer := httptest.NewServer(goproxy.NewServer(
		proxies.MultiChainEthereumRPC(getRpcMap()),
	))
	defer rpcServer.Close()

	rpcServerClient := rpcServer.Client()

	for chainID := range getRpcMap() {
		res, err := sendRequest[string](rpcServerClient, rpcServer.URL, http.MethodPost, rpcRequest{
			ID:      0,
			JSONRPC: "2.0",
			Method:  "eth_chainId",
			ChainID: chainID,
		})

		assert.NoError(t, err)

		chainIDResult, ok := big.NewInt(0).SetString(res.Result, 0)
		assert.True(t, ok)
		assert.Equal(t, int64(chainID), chainIDResult.Int64())
	}
}

func TestMultiChainEthereumRPCUnsupportedChain(t *testing.T) {
	rpcServer := httptest.NewServer(goproxy.NewServer(
		proxies.MultiChainEthereumRPC(map[int]string{}),
	))
	defer rpcServer.Close()

	rpcServerClient := rpcServer.Client()

	res, err := sendRequest[string](rpcServerClient, rpcServer.URL, http.MethodPost, rpcRequest{
		ID:      0,
		JSONRPC: "2.0",
		Method:  "eth_chainId",
		ChainID: 1,
	})
	assert.NoError(t, err)

	assert.Equal(t, "", res.Result)
	assert.Equal(t, -32000, res.Error.Code)
	assert.Equal(t, "Unsupported chain ID: 1", res.Error.Message)
}

func TestMultiChainEthereumRPCMethodNotAllowed(t *testing.T) {
	rpcServer := httptest.NewServer(goproxy.NewServer(
		proxies.MultiChainEthereumRPC(getRpcMap()),
	))
	defer rpcServer.Close()

	rpcServerClient := rpcServer.Client()

	res, err := sendRequest[string](rpcServerClient, rpcServer.URL, http.MethodGet, rpcRequest{})
	assert.NoError(t, err)

	assert.Equal(t, "", res.Result)
	assert.Equal(t, -32000, res.Error.Code)
	assert.Equal(t, "method not allowed", res.Error.Message)
}

func TestMultiChainEthereumRPCUseQuery(t *testing.T) {
	rpcServer := httptest.NewServer(goproxy.NewServer(
		proxies.MultiChainEthereumRPC(getRpcMap(), "chainId"),
	))
	defer rpcServer.Close()

	rpcServerClient := rpcServer.Client()

	for chainID := range getRpcMap() {
		res, err := sendRequest[string](rpcServerClient, fmt.Sprintf("%s?chainId=%d", rpcServer.URL, chainID), http.MethodPost, rpcRequest{
			ID:      0,
			JSONRPC: "2.0",
			Method:  "eth_chainId",
		})

		assert.NoError(t, err)

		chainIDResult, ok := big.NewInt(0).SetString(res.Result, 0)
		assert.True(t, ok)
		assert.Equal(t, int64(chainID), chainIDResult.Int64())
	}
}

func TestSingleRPC(t *testing.T) {
	targetURL, _ := url.Parse("https://api.avax.network/ext/bc/C/rpc")
	rpcServer := httptest.NewServer(goproxy.NewServer(
		goproxy.WithTargetURL(targetURL),
	))
	defer rpcServer.Close()

	rpcServerClient := rpcServer.Client()

	res, err := sendRequest[string](rpcServerClient, rpcServer.URL, http.MethodPost, rpcRequest{
		ID:      0,
		JSONRPC: "2.0",
		Method:  "eth_chainId",
	})

	assert.NoError(t, err)

	chainIDResult, ok := big.NewInt(0).SetString(res.Result, 0)
	assert.True(t, ok)
	assert.Equal(t, int64(43114), chainIDResult.Int64())
}
