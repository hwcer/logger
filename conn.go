package logger

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
)

func NewConn(network, address string) *Conn {
	return &Conn{Network: network, Address: address}
}

type Conn struct {
	sync.Mutex
	Network     string `json:"network"`
	Address     string `json:"address"`
	Reconnect   bool   `json:"reconnect"`
	Format      func(*Message) string
	innerWriter io.WriteCloser
	illNetFlag  bool //网络异常标记
}

func (c *Conn) Name() string {
	return c.Network + "://" + c.Address
}

func (c *Conn) Init() error {
	c.Lock()
	defer c.Unlock()
	if c.innerWriter != nil {
		_ = c.innerWriter.Close()
		c.innerWriter = nil
	}
	return nil
}

// Write 实现Output接口;连接状态与网络写入全程持锁,避免并发写与重复重连
func (c *Conn) Write(msg *Message) {
	c.Lock()
	defer c.Unlock()
	if c.needToConnectOnMsg() {
		if err := c.connect(); err != nil {
			return
		}
		c.illNetFlag = false
	}
	if !c.illNetFlag {
		//网络异常时跳过写入;写入出错时置位illNetFlag等待下次重连
		if err := c.println(msg); err != nil {
			c.illNetFlag = true
		}
	}
}

func (c *Conn) Close() error {
	c.Lock()
	defer c.Unlock()
	if c.innerWriter != nil {
		_ = c.innerWriter.Close()
		c.innerWriter = nil
	}
	return nil
}

func (c *Conn) connect() error {
	if c.innerWriter != nil {
		_ = c.innerWriter.Close()
		c.innerWriter = nil
	}
	addrs := strings.SplitSeq(c.Address, ";")
	for addr := range addrs {
		conn, err := net.Dial(c.Network, addr)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "net.Dial error:%v\n", err)
			continue
			//return err
		}

		if tcpConn, ok := conn.(*net.TCPConn); ok {
			_ = tcpConn.SetKeepAlive(true)
		}
		c.innerWriter = conn
		return nil
	}
	return fmt.Errorf("hava no valid logs service addr:%v", c.Address)
}

func (c *Conn) needToConnectOnMsg() bool {
	if c.Reconnect {
		c.Reconnect = false
		return true
	}

	if c.innerWriter == nil {
		return true
	}

	if c.illNetFlag {
		return true
	}
	return false
	//return c.Options.ReconnectOnMsg
}

// println 写入一条日志,调用方必须已持有锁
func (c *Conn) println(msg *Message) (err error) {
	var txt string
	if c.Format != nil {
		txt = c.Format(msg)
	} else {
		txt = msg.Content
	}
	if msg.Level >= LevelError {
		txt = txt + "\n" + msg.Stack
	}
	_, err = c.innerWriter.Write(append([]byte(txt), '\n'))
	return err
}
