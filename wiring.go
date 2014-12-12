package main

import (
	"log"
	"net"
)

type Wiring map[string]string

func (w *Wiring) Dial(network, address string) (net.Conn, error) {
	n := try(w.Route(address))
	if n != address {
		log.Printf("Routed %s -> %s\n", address, n)
	}
	return net.Dial(network, n)
}

func (w Wiring) RouteHost(host string, port string) (string, error) {
	address := net.JoinHostPort(host, port)
	if n, ok := w[address]; ok {
		h, _ := try(net.SplitHostPort(n))
		return h, nil
	}
	if n, ok := w[host]; ok {
		host = n
	}
	return host, nil
}

func (w Wiring) Route(address string) (string, error) {
	if n, ok := w[address]; ok {
		return n, nil
	}
	host, port := try(net.SplitHostPort(address))
	host = try(w.RouteHost(host, port))
	return net.JoinHostPort(host, port), nil
}

func Dial(network, address string) (net.Conn, error) {
	return state.Config.Wiring.Dial(network, address)
}
