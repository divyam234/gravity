package aria2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Client struct {
	url     string
	secret  string
	httpUrl string
}

func NewClient(wsUrl, secret string) *Client {
	httpUrl := wsUrl
	if bytes.HasPrefix([]byte(wsUrl), []byte("ws://")) {
		httpUrl = "http://" + wsUrl[5:]
	} else if bytes.HasPrefix([]byte(wsUrl), []byte("wss://")) {
		httpUrl = "https://" + wsUrl[6:]
	}

	return &Client{
		url:     wsUrl,
		secret:  secret,
		httpUrl: httpUrl,
	}
}

type JsonRpcRequest struct {
	JsonRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Id      string      `json:"id"`
	Params  interface{} `json:"params"`
}

type JsonRpcResponse struct {
	Result json.RawMessage `json:"result,omitempty"`
	Error  *JsonRpcError   `json:"error,omitempty"`
	Id     interface{}     `json:"id"`
}

type JsonRpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (c *Client) Call(ctx context.Context, method string, params ...interface{}) (json.RawMessage, error) {
	finalParams := []interface{}{"token:" + c.secret}
	finalParams = append(finalParams, params...)

	reqBody := JsonRpcRequest{
		JsonRPC: "2.0",
		Method:  method,
		Id:      fmt.Sprintf("%d", time.Now().UnixNano()),
		Params:  finalParams,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.httpUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rpcResp JsonRpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, err
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("aria2 rpc error: %d %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

type Notification struct {
	Method string `json:"method"`
	Params []struct {
		Gid string `json:"gid"`
	} `json:"params"`
}

func (c *Client) Listen(ctx context.Context, handler func(method string, gid string)) error {
	conn, _, err := websocket.Dial(ctx, c.url, nil)
	if err != nil {
		return err
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	for {
		var msg Notification
		if err := wsjson.Read(ctx, conn, &msg); err != nil {
			return err
		}

		if len(msg.Params) > 0 {
			handler(msg.Method, msg.Params[0].Gid)
		}
	}
}
