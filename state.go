package main

import (
	openssl "github.com/lunixbochs/go-openssl"
	"net"
)

type State struct {
	DataDir  string
	CAKey    openssl.PrivateKey
	CA       *openssl.Certificate
	Sessions map[*Session]int
	Listen   string
}

var state *State = &State{
	Sessions: make(map[*Session]int),
}

func (s *State) OnConnect(conn net.Conn) {
	c := NewSession(conn)
	s.Sessions[c] = 1
	go c.Handle()
}

func (s *State) OnDisconnect(c *Session) {
	// when would this get fired? by the session I assume.
	delete(s.Sessions, c)
}
