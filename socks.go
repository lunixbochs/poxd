package main

import (
	"bytes"
	"errors"
	"log"
	"net"
	"strconv"
)

const (
	NullAuth         = 0x00
	GSSAPIAuth       = 0x01
	UserPassAuth     = 0x02
	NoAcceptableAuth = 0xFF
)

type Socks struct {
	Conn
	Remote string
}

func WrapSocks(c Conn) *Socks {
	return &Socks{
		Conn: c,
	}
}

func IsSocks(c Conn) bool {
	p, err := c.Peek(2)
	if err != nil {
		return false
	}
	return p[0] == 0x05 && p[1] > 0
}

func (c *Socks) Handshake() error {
	write := func(p []byte) (int, error) {
		p = append([]byte{0x05}, p...)
		return c.Conn.Write(p)
	}
	ver := try(c.ReadByte())
	if ver != 0x05 {
		return errors.New("SOCKS version != 5")
	}
	nMethods := try(c.ReadByte())
	if nMethods < 1 {
		return errors.New("No auth methods offered")
	}
	var methods []byte
	for ; nMethods > 0; nMethods-- {
		method := try(c.ReadByte())
		methods = append(methods, method)
	}
	// only NULL auth is accepted
	if bytes.Contains(methods, []byte{NullAuth}) {
		_ = try(write([]byte{NullAuth}))
	} else {
		_ = try(write([]byte{NoAcceptableAuth}))
		return errors.New("No acceptable auth methods")
	}

	ver = try(c.ReadByte())
	if ver != 0x05 {
		return errors.New("SOCKS version != 5")
	}
	cmd := try(c.ReadByte())
	rsv := try(c.ReadByte())
	if rsv != 0 {
		return errors.New("Reserved byte must be 0")
	}
	addrType := try(c.ReadByte())
	addr := try(c.recvAddr(addrType))

	// var remote net.Conn
	code := byte(0)
	switch cmd {
	case 0x01:
		// CONNECT
		// Connect always "succeeds", but actually happens later.
		c.Remote = addr
		// on error:
		// code = 0x5
	default:
		// we got either a BIND or a UDP ASSOCIATE request
		// we don't support those (yet?)
		code = 0x7
	}
	reply := []byte{code, 0x00, 0x01}
	if code == 0 {
		// TODO: actual bound ATYP; we here assume IPv4
		// the RFC is unclear on whether this actually matters for CONNECT
		// it doesn't seem to

		// bindHost, bindPortStr := try(net.SplitHostPort(remote.LocalAddr().String()))
		// deal with it...
		bindHost, bindPortStr := "0.0.0.0", "0"
		bindIPBytes := []byte(net.ParseIP(bindHost))[12:]
		bindPort := try(strconv.Atoi(bindPortStr))

		bindPortBytes := []byte{byte((bindPort & 0xFF00) >> 8), byte(bindPort & 0xFF)}
		reply = append(reply, bindIPBytes...)
		reply = append(reply, bindPortBytes...)
	}
	write(reply)
	if code > 0 {
		c.Close()
	}
	return nil
}

func (c *Socks) recvAddr(addrType byte) (string, error) {
	var host string
	switch addrType {
	case 0x01:
		// IPv4
		rawIp := make([]byte, 4)
		length := try(c.Read(rawIp))
		if length != 4 {
			return "", errors.New("IPv4 address was less than 4 bytes long")
		}
		host = net.IP(rawIp).String()
	case 0x03:
		// domain
		nameLen := try(c.ReadByte())
		nameBytes := make([]byte, nameLen)
		length := try(c.Read(nameBytes))
		if length != int(nameLen) {
			return "", errors.New("Failed to read entire domain name")
		}
		host = string(nameBytes)
	case 0x04:
		// IPv6
		rawIp := make([]byte, 16)
		length := try(c.Read(rawIp))
		if length != 16 {
			return "", errors.New("IPv6 address was less than 16 bytes long")
		}
		host = net.IP(rawIp).String()
	}
	portBytes := make([]byte, 2)
	length := try(c.Read(portBytes))
	if length != 2 {
		return "", errors.New("Port number must be 2 bytes long")
	}
	port := (int(portBytes[0]) << 8) + int(portBytes[1])
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	return addr, nil
}

func (c *Socks) Connect() (net.Conn, error) {
	return net.Dial("tcp", c.Remote)
}

func (c *Socks) Proxy() error {
	remote := try(c.Connect())
	Pipe(c, remote)
	return nil
}
