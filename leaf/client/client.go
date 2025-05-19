package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	serverAddr     = "127.0.0.1:3563"
	readBufferSize = 1024
	writeInterval  = time.Second

	// 新增配置常量
	dialTimeout     = 5 * time.Second
	readTimeout     = 3 * time.Second
	heartbeatPeriod = 30 * time.Second
	maxRetries      = 3
)

type Client struct {
	conn    net.Conn
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	msgPool sync.Pool
}

func NewClient() *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		ctx:    ctx,
		cancel: cancel,
		msgPool: sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}
}

func (c *Client) Connect() error {
	dialer := net.Dialer{Timeout: dialTimeout}
	var err error
	for i := 0; i < maxRetries; i++ {
		c.conn, err = dialer.DialContext(c.ctx, "tcp", serverAddr)
		if err == nil {
			return nil
		}
		time.Sleep(time.Second * time.Duration(i+1))
	}
	return fmt.Errorf("连接失败，已重试%d次: %w", maxRetries, err)
}

func (c *Client) Run() {
	if err := c.Connect(); err != nil {
		fmt.Printf("连接服务器失败: %v\n", err)
		return
	}
	defer c.conn.Close()

	c.wg.Add(3) // 增加心跳goroutine

	// 写入goroutine
	go func() {
		defer c.wg.Done()
		if err := c.write(); err != nil {
			fmt.Printf("写入错误: %v\n", err)
			c.cancel()
		}
	}()

	// 读取goroutine
	go func() {
		defer c.wg.Done()
		if err := c.read(); err != nil {
			fmt.Printf("读取错误: %v\n", err)
			c.cancel()
		}
	}()

	// 心跳goroutine
	go func() {
		defer c.wg.Done()
		c.heartbeat()
	}()

	c.wg.Wait()
	fmt.Println("客户端已退出")
}

func (c *Client) heartbeat() {
	ticker := time.NewTicker(heartbeatPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			fmt.Println("ctx.Done 心跳退出")
			return
		case <-ticker.C:
			msg := &Wrapper{
				Hello: Hello{
					Name: "heartbeat",
				},
			}
			if err := c.sendMessage(msg); err != nil {
				fmt.Printf("发送心跳失败: %v\n", err)
				c.cancel()
				return
			}
		}
	}
}

func (c *Client) write() error {
	ticker := time.NewTicker(writeInterval)
	defer ticker.Stop()
	var i int
	for {
		select {
		case <-c.ctx.Done():
			fmt.Println("ctx.Done write退出")
			return nil
		case <-ticker.C:
			w := &Wrapper{
				Hello: Hello{
					Name: fmt.Sprintf("leaf-%d", i),
				},
			}
			if err := c.sendMessage(w); err != nil {
				return err
			}

			// // todo delete debug code
			// if i >= 10 {
			// 	return fmt.Errorf("发送消息数量超过10")
			// }
			i++
		}
	}
}

func (c *Client) sendMessage(msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("消息序列化失败: %w", err)
	}

	m := make([]byte, 2+len(data))
	binary.BigEndian.PutUint16(m, uint16(len(data)))
	copy(m[2:], data)

	_, err = c.conn.Write(m)
	if err != nil {
		return fmt.Errorf("发送消息失败: %w", err)
	}
	return nil
}

func (c *Client) read() error {
	buff := make([]byte, readBufferSize)

	for {
		select {
		case <-c.ctx.Done():
			fmt.Println("ctx.Done read退出")
			return nil
		default:
			if err := c.conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
				return fmt.Errorf("设置读取超时失败: %w", err)
			}

			msgbuff := c.msgPool.Get().(*bytes.Buffer)
			msgbuff.Reset()
			defer c.msgPool.Put(msgbuff)

			n, err := c.conn.Read(buff)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				return fmt.Errorf("读取数据失败: %w", err)
			}

			msgbuff.Write(buff[:n])
			if err := c.processMessages(msgbuff); err != nil {
				fmt.Printf("处理消息失败: %v\n", err)
			}
		}
	}
}

func (c *Client) processMessages(msgbuff *bytes.Buffer) error {
	for {
		if msgbuff.Len() < 2 {
			return nil
		}

		msgLen := int(binary.BigEndian.Uint16(msgbuff.Bytes()[:2])) + 2
		if msgbuff.Len() < msgLen {
			return nil
		}

		data := msgbuff.Next(msgLen)
		var tMsg Wrapper
		if err := json.Unmarshal(data[2:], &tMsg); err != nil {
			return fmt.Errorf("JSON解析错误: %w", err)
		}

		// 忽略心跳响应
		if !strings.Contains(tMsg.Hello.Name, "heartbeat") {
			fmt.Printf("收到消息: %s\n", tMsg.Hello.Name)
		}
	}
}

func main() {
	client := NewClient()
	client.Run()
}

type Wrapper struct {
	Hello Hello `json:"Hello"`
}

type Hello struct {
	Name string
}
