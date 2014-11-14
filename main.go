package main

import (
	"flag"
	cacert "github.com/lunixbochs/go-cacert"
	openssl "github.com/lunixbochs/go-openssl"
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
	flag.StringVar(&state.Listen, "listen", "localhost:1080", "proxy listen address")
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
	log.Println("Data path:    ", state.DataDir)
	log.Println("Web interface:", "http://"+state.Listen)
	log.Println()

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

	if _, err := os.Stat(rootCAPath); os.IsNotExist(err) {
		if err := os.Mkdir(path.Dir(rootCAPath), os.ModeDir|0700); err != nil && !os.IsExist(err) {
			log.Fatal(err)
		}
		log.Printf("Writing Mozilla trusted roots -> '%s'", rootCAPath)
		try(ioutil.WriteFile(rootCAPath, cacert.Bundle, 0700))
	}

	log.Println()
	ln := try(net.Listen("tcp", state.Listen))
	log.Println("Listening for connections.")
	log.Println()
	for {
		conn := try(ln.Accept())
		log.Println("Connection from:", conn.RemoteAddr())
		state.OnConnect(conn)
	}
}
