package main

import (
	"log"
	"net"
	"os"
	"path"

	openssl "github.com/lunixbochs/go-openssl"
)

type State struct {
	Config    *Config
	DataDir   string
	Listen    string
	ListenAlt string
	Sessions  map[*Session]int

	CA      *openssl.Certificate
	CAKey   openssl.PrivateKey
	RootCAs *openssl.CertificateStore
}

var state *State = &State{
	Sessions: make(map[*Session]int),
}

func (s *State) LoadConfig() (*Config, bool, error) {
	configPath := path.Join(s.DataDir, "config.yml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		s.Config = &Config{}
		return s.Config, true, nil
	} else if err != nil {
		return nil, false, err
	}
	s.Config = try(LoadConfig(configPath))
	return s.Config, false, nil
}

func (s *State) SaveConfig() error {
	configPath := path.Join(s.DataDir, "config.yml")
	return s.Config.Save(configPath)
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
