package aria2

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"gravity/internal/logger"

	"go.uber.org/zap"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Client struct {
	wsUrl string
	conn  *websocket.Conn
	mu    sync.Mutex
	// Map to store pending requests: ID -> channel
	pending map[string]chan *JsonRpcResponse
	// Callback for notifications
	onNotification func(method string, params []interface{})
}

func NewClient(wsUrl string) *Client {
	return &Client{
		wsUrl:   wsUrl,
		pending: make(map[string]chan *JsonRpcResponse),
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
	Method string          `json:"method,omitempty"`
	Params []interface{}   `json:"params,omitempty"`
}

type JsonRpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return nil
	}

	conn, _, err := websocket.Dial(ctx, c.wsUrl, nil)
	if err != nil {
		return err
	}
	c.conn = conn

	// Start listening loop
	go c.listen()

	return nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return c.conn.Close(websocket.StatusNormalClosure, "closing")
	}
	return nil
}

func (c *Client) SetNotificationHandler(handler func(method string, params []interface{})) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onNotification = handler
}

func (c *Client) listen() {
	for {
		var resp JsonRpcResponse
		if c.conn == nil {
			return
		}

		err := wsjson.Read(context.Background(), c.conn, &resp)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure || websocket.CloseStatus(err) == websocket.StatusGoingAway {
				// Normal closure
			} else {
				logger.L.Warn("Aria2 websocket read error", zap.Error(err))
			}
			c.mu.Lock()
			c.conn = nil
			c.mu.Unlock()
			return
		}

		// Handle Notification
		if resp.Id == nil && resp.Method != "" {
			c.mu.Lock()
			handler := c.onNotification
			c.mu.Unlock()
			if handler != nil {
				go handler(resp.Method, resp.Params)
			}
			continue
		}

		// Handle Response
		if resp.Id != nil {
			idStr := fmt.Sprintf("%v", resp.Id)
			c.mu.Lock()
			if ch, ok := c.pending[idStr]; ok {
				ch <- &resp
				delete(c.pending, idStr)
			}
			c.mu.Unlock()
		}
	}
}

func (c *Client) Call(ctx context.Context, method string, params ...interface{}) (json.RawMessage, error) {
	// Ensure connected
	if c.conn == nil {
		if err := c.Connect(ctx); err != nil {
			return nil, fmt.Errorf("failed to connect to aria2: %w", err)
		}
	}

	// For internal use, we pass empty params if none provided
	p := params
	if len(p) == 0 {
		p = []interface{}{}
	}

	id := fmt.Sprintf("%d", rand.Int63())
	reqBody := JsonRpcRequest{
		JsonRPC: "2.0",
		Method:  method,
		Id:      id,
		Params:  p,
	}

	respCh := make(chan *JsonRpcResponse, 1)
	c.mu.Lock()
	c.pending[id] = respCh
	c.mu.Unlock()

	if err := wsjson.Write(ctx, c.conn, reqBody); err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, err
	}

	select {
	case resp := <-respCh:
		if resp.Error != nil {
			return nil, fmt.Errorf("aria2 rpc error: %d %s", resp.Error.Code, resp.Error.Message)
		}
		return resp.Result, nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, ctx.Err()
	case <-time.After(10 * time.Second):
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("aria2 rpc timeout")
	}
}
