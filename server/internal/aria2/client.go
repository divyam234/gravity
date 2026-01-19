package aria2

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Client struct {
	url        string
	secret     string
	httpUrl    string
	onStart    func(gid string)
	onPause    func(gid string)
	onStop     func(gid string)
	onComplete func(gid string)
	onError    func(gid string)
}

func NewClient(wsUrl, secret string) *Client {
	// Derive HTTP URL from WS URL for RPC calls
	httpUrl := strings.Replace(wsUrl, "ws://", "http://", 1)
	httpUrl = strings.Replace(httpUrl, "wss://", "https://", 1)

	return &Client{
		url:     wsUrl,
		secret:  secret,
		httpUrl: httpUrl,
	}
}

func (c *Client) SetOnStartHandler(handler func(gid string)) {
	c.onStart = handler
}

func (c *Client) SetOnPauseHandler(handler func(gid string)) {
	c.onPause = handler
}

func (c *Client) SetOnStopHandler(handler func(gid string)) {
	c.onStop = handler
}

func (c *Client) SetOnCompleteHandler(handler func(gid string)) {
	c.onComplete = handler
}

func (c *Client) SetOnErrorHandler(handler func(gid string)) {
	c.onError = handler
}

type Notification struct {
	Method string `json:"method"`
	Params []struct {
		Gid string `json:"gid"`
	} `json:"params"`
}

type JsonRpcRequest struct {
	JsonRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Id      string      `json:"id"`
	Params  interface{} `json:"params"`
}

type JsonRpcResponse struct {
	Result interface{} `json:"result"`
	Error  interface{} `json:"error"`
}

func (c *Client) Call(method string, params ...interface{}) (interface{}, error) {
	// Inject secret if needed
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

	req, err := http.NewRequest("POST", c.httpUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	// Explicitly request no compression to avoid gzip issues
	req.Header.Set("Accept-Encoding", "identity")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Handle potential gzip response (in case server ignores Accept-Encoding)
	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("gzip decode error: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	var rpcResp JsonRpcResponse
	if err := json.NewDecoder(reader).Decode(&rpcResp); err != nil {
		return nil, err
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC Error: %v", rpcResp.Error)
	}

	return rpcResp.Result, nil
}

func (c *Client) TellStatus(gid string) (map[string]interface{}, error) {
	res, err := c.Call("aria2.tellStatus", gid)
	if err != nil {
		return nil, err
	}
	return res.(map[string]interface{}), nil
}

func (c *Client) GetVersion() (map[string]interface{}, error) {
	res, err := c.Call("aria2.getVersion")
	if err != nil {
		return nil, err
	}
	return res.(map[string]interface{}), nil
}

func (c *Client) Listen(ctx context.Context) error {
	for {
		err := c.connectAndListen(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			log.Printf("Aria2 WS error: %v. Retrying in 5s...", err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
				continue
			}
		}
	}
}

func (c *Client) connectAndListen(ctx context.Context) error {
	conn, _, err := websocket.Dial(ctx, c.url, nil)
	if err != nil {
		return err
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	log.Println("Connected to Aria2 WebSocket")

	for {
		var msg Notification
		if err := wsjson.Read(ctx, conn, &msg); err != nil {
			return err
		}

		if len(msg.Params) > 0 {
			gid := msg.Params[0].Gid

			switch msg.Method {
			case "aria2.onDownloadStart":
				log.Printf("Download Start: %s", gid)
				if c.onStart != nil {
					go c.onStart(gid)
				}
			case "aria2.onDownloadPause":
				log.Printf("Download Paused: %s", gid)
				if c.onPause != nil {
					go c.onPause(gid)
				}
			case "aria2.onDownloadStop":
				log.Printf("Download Stopped: %s", gid)
				if c.onStop != nil {
					go c.onStop(gid)
				}
			case "aria2.onDownloadComplete":
				log.Printf("Download Complete: %s", gid)
				if c.onComplete != nil {
					go c.onComplete(gid)
				}
			case "aria2.onDownloadError":
				log.Printf("Download Error: %s", gid)
				if c.onError != nil {
					go c.onError(gid)
				}
			}
		}
	}
}
