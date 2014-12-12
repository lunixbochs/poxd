package main

import (
	cacert "github.com/lunixbochs/go-cacert"
	openssl "github.com/lunixbochs/go-openssl"
	"github.com/lunixbochs/redigo/redis"

	"crypto/rand"
	"encoding/hex"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/user"
	"path"
)

func main() {
	user := try(user.Current())
	dir := flag.String("base", user.HomeDir, "directory containing .poxd folder")
	flag.StringVar(&state.Listen, "listen", ":1080", "proxy listen address")
	flag.StringVar(&state.ListenAlt, "listen_alt", ":1081", "proxy listen address (not logged)")
	flag.StringVar(&state.Redis, "redis", ":6379", "redis server address")
	flag.Parse()

	// ensure data directory exists and is sane
	state.DataDir = path.Join(*dir, ".poxd")
	err = os.Mkdir(state.DataDir, os.ModeDir|0700)
	if os.IsExist(err) {
		stat := try(os.Lstat(state.DataDir))
		if !stat.IsDir() || (stat.Mode()&os.ModeSymlink) != 0 {
			log.Fatalf("%s does not appear to be a directory", state.DataDir)
		}
		if (stat.Mode().Perm() & 0007) != 0 {
			log.Fatalf("Refusing to trust world-accessible %s", state.DataDir)
		}
	} else if err != nil {
		log.Fatal(err)
	}

	log.Println("Bind address: ", state.Listen)
	log.Println("Bind address: ", state.ListenAlt, "(not logged)")
	log.Println("Data path:    ", state.DataDir)
	log.Println("Web interface:", "http://"+state.Listen)
	log.Println()

	// load or generate CA and private key
	keyPath := path.Join(state.DataDir, "ca", "ca.key")
	caPath := path.Join(state.DataDir, "ca", "ca.crt")
	rootCAPath := path.Join(state.DataDir, "certs", "roots.pem")
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		log.Printf("Generating 2048-bit RSA private key -> '%s'", keyPath)
		try(MakeRSAKey(keyPath))
		if err := os.Remove(caPath); err != nil && !os.IsNotExist(err) {
			log.Fatal(err)
		}
	}
	key := try(ioutil.ReadFile(keyPath))
	state.CAKey = try(openssl.LoadPrivateKeyFromPEM(key))

	if _, err := os.Stat(caPath); os.IsNotExist(err) {
		log.Printf("Generating new SSL CA -> '%s'", caPath)
		try(MakeCA(caPath))
		log.Println("NOTE: A new CA has been generated. Add it to your keychain for interception to succeed.")
	}
	ca := try(ioutil.ReadFile(caPath))
	state.CA = try(openssl.LoadCertificateFromPEM(ca))

	// we default to our own copy of mozilla's trusted root certificates
	if _, err := os.Stat(rootCAPath); os.IsNotExist(err) {
		if err := os.Mkdir(path.Dir(rootCAPath), os.ModeDir|0700); err != nil && !os.IsExist(err) {
			log.Fatal(err)
		}
		log.Printf("Writing Mozilla trusted roots -> '%s'", rootCAPath)
		try(ioutil.WriteFile(rootCAPath, cacert.Bundle, 0700))
	}

	state.RootCAs = try(LoadRootCAs(rootCAPath))

	// load/check config and print useful values
	config, first := try(state.LoadConfig())
	if first {
		log.Println("Generating first API key.")
		binKey := make([]byte, 16)
		_ = try(rand.Read(binKey))
		apiKey := hex.EncodeToString(binKey)
		config.ApiKeys = append(config.ApiKeys, apiKey)
		try(state.SaveConfig())
	}
	if len(config.ApiKeys) > 0 {
		log.Println("API keys:")
		for _, v := range config.ApiKeys {
			log.Printf("  - %s\n", v)
		}
	}

	// try connecting to Redis
	client, err := redis.Dial("tcp", state.Redis)
	redisWorks := true
	if err != nil {
		redisWorks = false
		log.Println("Could not connect to Redis: ", err)
	} else {
		client.Close()
	}

	// time to start
	loggedAccept := make(chan net.Conn)
	nologAccept := make(chan net.Conn)
	acceptFunc := func(ret chan net.Conn, ln net.Listener) {
		for {
			conn := try(ln.Accept())
			ret <- conn
		}
	}

	log.Println()
	ln1 := try(net.Listen("tcp", state.Listen))
	ln2 := try(net.Listen("tcp", state.ListenAlt))
	go acceptFunc(loggedAccept, ln1)
	go acceptFunc(nologAccept, ln2)
	log.Println("Listening for connections.")
	log.Println()
	for {
		var conn net.Conn
		var logged bool
		select {
		case conn = <-loggedAccept:
			if redisWorks {
				logged = true
				log.Printf("(logging) ")
			}
		case conn = <-nologAccept:
			logged = false
		}
		log.Println("Connection from:", conn.RemoteAddr())
		state.OnConnect(conn, logged)
	}
}
