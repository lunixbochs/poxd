package main

import (
	"log"
	"net"
)

type Session struct {
	Conn
	chain     []Conn
	ShouldLog bool
}

func NewSession(conn net.Conn, logged bool) *Session {
	bf := NewConn(conn)
	s := &Session{Conn: bf, ShouldLog: logged}
	s.chain = append(s.chain, bf)
	return s
}

func (s *Session) Chain(c Conn) Conn {
	s.chain = append(s.chain, c)
	s.Conn = c
	return c
}

func (s *Session) Handle() {
	if IsSocks(s) {
		c := WrapSocks(s.Conn)
		s.Chain(c)
		err := c.Handshake()
		if err != nil {
			c.Close()
			return
		}
		log.Println("CONNECT:", c.Remote)
		if IsTLS(s) {
			log.Println("TLS detected.")
			remote, err := c.Connect()
			if err != nil {
				s.Close()
				return
			}
			remote = WrapTLSServer(remote)
			host, _ := try(net.SplitHostPort(c.Remote))
			tls := WrapTLSClient(s.Conn, host)
			s.Chain(NewConn(tls))

			if IsHttp(s) && s.ShouldLog {
				LogHttp(s, remote)
			} else {
				Pipe(s, remote)
			}
		} else {
			if IsHttp(s) && s.ShouldLog {
				remote, err := c.Connect()
				if err != nil {
					s.Close()
					return
				}
				LogHttp(s, remote)
			} else {
				c.Proxy()
			}
		}
		return
	} else {
		if IsHttp(s) {
			log.Println("Detected HTTP.")
			s.Close()
			return
		}
	}
	log.Println("SOCKS not detected.")
	s.Close()
}
