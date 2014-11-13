package main

import (
	"bufio"
	"io"
	"net"
)

type BufReader interface {
	Buffered() int
	Peek(n int) ([]byte, error)
	ReadByte() (byte, error)
	ReadBytes(byte) ([]byte, error)
	ReadLine() ([]byte, bool, error)
	/*
	        // We don't need these, even though they're embedded by bufio.Reader
			ReadRune() (rune, int, error)
			ReadSlice(byte) ([]byte, error)
			ReadString(byte) (string, error)
			Reset(io.Reader)
			UnreadByte() error
			UnreadRune() error
	*/
	WriteTo(io.Writer) (int64, error)
}

type Conn interface {
	net.Conn
	BufReader
}

type BufConn struct {
	net.Conn
	*bufio.Reader
}

func NewConn(conn net.Conn) Conn {
	return &BufConn{conn, bufio.NewReader(conn)}
}

func (b *BufConn) Read(p []byte) (int, error) {
	return b.Reader.Read(p)
}
