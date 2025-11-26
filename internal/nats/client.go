package nats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

type Message struct {
	RequestID string      `json:"requestid"`
	Cmd       string      `json:"cmd"`
	Meta      interface{} `json:"meta"`
	Payload   interface{} `json:"payload"`
}

type ResponseMessage struct {
	RequestID string      `json:"requestid"`
	Meta      interface{} `json:"meta"`
	Payload   interface{} `json:"payload"`
}

type Client struct {
	nc *nats.Conn
}

func NewClient(url string) (*Client, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}
	return &Client{nc: nc}, nil
}

func (c *Client) Request(ctx context.Context, subject string, req Message) (*ResponseMessage, error) {

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	replyTo := fmt.Sprintf("reply.%s", req.RequestID)

	sub, err := c.nc.SubscribeSync(replyTo)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to reply: %w", err)
	}
	defer sub.Unsubscribe()

	err = c.nc.PublishRequest(subject, replyTo, reqBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to publish request: %w", err)
	}

	msg, err := sub.NextMsg(5 * time.Second) // Таймаут 5 секунд
	if err != nil {
		return nil, errors.New("timeout waiting for response")
	}

	var resp ResponseMessage
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// Close закрывает соединение
func (c *Client) Close() {
	if c.nc != nil {
		c.nc.Close()
	}
}
