package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

func main() {
	Run()
}

func Run() {
	conn, err := net.Dial("tcp", "127.0.0.1:3563")
	if err != nil {
		panic(err)
	}

	go write(conn)
	err = read(conn)
	fmt.Println("exit: read error:", err)
}

type Wrapper struct {
	Hello Hello `json:"Hello"`
}

type Hello struct {
	Name string
}

func write(conn net.Conn) {
	h := Hello{
		Name: "leaf",
	}
	w := &Wrapper{
		Hello: h,
	}

	var i int
	for {
		w.Hello.Name = fmt.Sprintf("leaf-%d", i)
		data, _ := json.Marshal(w)

		m := make([]byte, 2+len(data))
		binary.BigEndian.PutUint16(m, uint16(len(data)))
		copy(m[2:], data)

		// 发送消息
		conn.Write(m)

		time.Sleep(time.Second * 1)
		i++
	}
}

func read(conn net.Conn) (err error) {
	buff := make([]byte, 1024)
	msgbuff := bytes.Buffer{} // 使用 bytes.Buffer 替代 []byte

	for {
		k, err := conn.Read(buff)
		if err != nil {
			return err
		}

		msgbuff.Write(buff[:k]) // 追加到缓冲区

		for {
			if msgbuff.Len() < 2 {
				break // 不足消息头部
			}

			msgLen := int(binary.BigEndian.Uint16(msgbuff.Bytes()[:2])) + 2
			if msgbuff.Len() < msgLen {
				break // 消息不完整
			}

			data := msgbuff.Next(msgLen) // 提取完整消息
			fmt.Printf("DEBUG: msgLen=%d, readLen=%d, data=%q\n",
				msgLen, k, data[2:]) // 调试输出

			var tMsg Wrapper
			if err := json.Unmarshal(data[2:], &tMsg); err != nil {
				fmt.Printf("JSON parse error: %v\n", err)
				continue
			}

			fmt.Println("recv:", tMsg.Hello.Name)
		}
	}
}
