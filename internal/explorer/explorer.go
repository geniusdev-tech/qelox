// Package explorer provides high-level proxy methods for querying the go-quai node RPC.
package explorer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/zeus/qelox/internal/config"
	"github.com/zeus/qelox/internal/node"
)

// Explorer handles querying the local go-quai RPC.
type Explorer struct {
	cfg        *config.Config
	node       *node.Controller
	httpClient *http.Client
}

// New creates a new Explorer instance.
func New(cfg *config.Config, nc *node.Controller) *Explorer {
	return &Explorer{
		cfg:  cfg,
		node: nc,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:    10,
				IdleConnTimeout: 30 * time.Second,
			},
		},
	}
}

// Search detects the type of query and returns type + data.
func (e *Explorer) Search(query string) (string, interface{}, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "error", nil, fmt.Errorf("empty query")
	}

	// 1. Is it a block number? (decimal)
	// We handle integer parsing lightly here, if it matches purely digits we try block by number
	isNumber := true
	for _, c := range query {
		if c < '0' || c > '9' {
			isNumber = false
			break
		}
	}
	if isNumber {
		// Convert decimal to hex for RPC
		var dec uint64
		fmt.Sscanf(query, "%d", &dec)
		hexNum := fmt.Sprintf("0x%x", dec)
		block, err := e.GetBlockByNumber(hexNum)
		if err == nil && block != nil {
			return "block", block, nil
		}
	}

	// 2. Is it a hash?
	if strings.HasPrefix(strings.ToLower(query), "0x") {
		// Length 66 = Transaction or Block Hash
		if len(query) == 66 {
			// Try TX first
			tx, err := e.GetTransaction(query)
			if err == nil && tx != nil {
				return "tx", tx, nil
			}
			// Try block by hash
			block, err := e.GetBlockByHash(query)
			if err == nil && block != nil {
				return "block", block, nil
			}
		}

		// Length 42 = Address
		if len(query) == 42 {
			addrData, err := e.GetAddressInfo(query)
			if err != nil {
				return "address", nil, err
			}
			return "address", addrData, nil
		}

	}

	return "unknown", nil, fmt.Errorf("could not find data for query: %s", query)
}

// GetBlockByHash fetches a block by its hash
func (e *Explorer) GetBlockByHash(hash string) (interface{}, error) {
	return e.rpcCall("quai_getBlockByHash", []interface{}{hash, true})
}

// GetBlockByNumber fetches a block by its number (hex string)
func (e *Explorer) GetBlockByNumber(hexNumber string) (interface{}, error) {
	return e.rpcCall("quai_getBlockByNumber", []interface{}{hexNumber, true})
}

// GetTransaction fetches a transaction and its receipt
func (e *Explorer) GetTransaction(hash string) (interface{}, error) {
	tx, err := e.rpcCall("quai_getTransactionByHash", []interface{}{hash})
	if err != nil || tx == nil {
		return nil, err
	}

	receipt, _ := e.rpcCall("quai_getTransactionReceipt", []interface{}{hash})

	// Combine into a single map
	result := make(map[string]interface{})
	if m, ok := tx.(map[string]interface{}); ok {
		for k, v := range m {
			result[k] = v
		}
	}
	if m, ok := receipt.(map[string]interface{}); ok {
		result["receipt"] = m
	}

	return result, nil
}

// GetAddressInfo fetches balance and transaction count for an address
func (e *Explorer) GetAddressInfo(address string) (interface{}, error) {
	balance, err1 := e.rpcCall("quai_getBalance", []interface{}{address, "latest"})
	nonce, err2 := e.rpcCall("quai_getTransactionCount", []interface{}{address, "latest"})

	if err1 != nil {
		return nil, err1
	}

	return map[string]interface{}{
		"address": address,
		"balance": balance,
		"nonce":   nonce,
	}, err2
}

// rpcCall is a helper to make JSON-RPC calls to the local go-quai node.
func (e *Explorer) rpcCall(method string, params []interface{}) (interface{}, error) {
	if !e.node.IsRunning() {
		return nil, fmt.Errorf("node is not running")
	}

	if params == nil {
		params = []interface{}{}
	}

	body, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      time.Now().UnixNano(),
	})

	resp, err := e.httpClient.Post(e.cfg.Monitor.RPCURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rpc http status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rpcResp struct {
		Result interface{} `json:"result"`
		Error  interface{} `json:"error"`
	}

	if err := json.Unmarshal(data, &rpcResp); err != nil {
		return nil, err
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("rpc error: %v", rpcResp.Error)
	}

	return rpcResp.Result, nil
}
