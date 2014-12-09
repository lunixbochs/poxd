package main

import (
	openssl "github.com/lunixbochs/go-openssl"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
	"time"
)

func LoadRootCAs(rootPath string) (*openssl.CertificateStore, error) {
	log.Println("Loading root CAs.")
	data := try(ioutil.ReadFile(rootPath))
	store := try(openssl.NewCertificateStore())
	try(store.LoadCertificatesFromPEM(data))
	return store, nil
}

func MakeRSAKey(keyPath string) error {
	key := try(openssl.GenerateRSAKey(2048))
	pubKey := try(key.MarshalPKIXPublicKeyPEM())
	privKey := try(key.MarshalPKCS1PrivateKeyPEM())
	err = os.Mkdir(path.Dir(keyPath), os.ModeDir|0700)
	if err != nil && !os.IsExist(err) {
		return err
	}
	try(ioutil.WriteFile(keyPath+".pub", pubKey, 0400))
	try(ioutil.WriteFile(keyPath, privKey, 0400))
	return nil
}

func MakeCA(caPath string) error {
	err = os.Mkdir(path.Dir(caPath), os.ModeDir|0700)
	if !os.IsExist(err) {
		return err
	}
	info := &openssl.CertificateInfo{
		Serial:       1,
		Issued:       0,
		Expires:      10 * 365 * 24 * time.Hour,
		Country:      "US",
		Organization: "poxd Root CA",
		CommonName:   "poxd Root CA",
	}
	ca := try(openssl.NewCertificate(info, state.CAKey))
	try(ca.AddExtensions(map[openssl.NID]string{
		openssl.NID_basic_constraints:      "critical,CA:TRUE",
		openssl.NID_key_usage:              "critical,keyCertSign,cRLSign",
		openssl.NID_subject_key_identifier: "hash",
		openssl.NID_netscape_cert_type:     "sslCA",
	}))
	try(ca.Sign(state.CAKey, openssl.EVP_SHA256))
	pem := try(ca.MarshalPEM())
	try(ioutil.WriteFile(caPath, pem, 0400))
	return nil
}

func MakeCert(hostname string) (*openssl.Certificate, error) {
	info := &openssl.CertificateInfo{
		Serial:       1,
		Issued:       0,
		Expires:      24 * time.Hour,
		Country:      "US",
		Organization: "poxd",
		CommonName:   hostname,
	}
	cert := try(openssl.NewCertificate(info, state.CAKey))
	try(cert.AddExtensions(map[openssl.NID]string{
		openssl.NID_basic_constraints: "critical,CA:FALSE",
		openssl.NID_key_usage:         "keyEncipherment",
		openssl.NID_ext_key_usage:     "serverAuth",
	}))
	try(cert.SetIssuer(state.CA))
	try(cert.Sign(state.CAKey, openssl.EVP_SHA256))
	return cert, nil
}

func IsTLS(c Conn) bool {
	p, err := c.Peek(6)
	if err != nil {
		return false
	}
	// leading byte == 22, version > 3.0, length > 20, message type = ClientHello
	return (len(p) > 5 && p[0] == 22 &&
		p[1] >= 3 && (p[3] > 0 || p[4] > 20) &&
		p[5] == 1)
}

func WrapTLSServer(c net.Conn) net.Conn {
	ctx := try(openssl.NewCtx())
	conn := try(openssl.Client(c, ctx))
	return conn
}

func WrapTLSClient(c net.Conn, hostname string) net.Conn {
	log.Println("Wrapping TLS with hostname:", hostname)
	ctx := try(openssl.NewCtx())
	cert := try(MakeCert(hostname))
	ctx.UseCertificate(cert)
	ctx.UsePrivateKey(state.CAKey)

	ctx.SetTLSExtServernameCallback(func(ssl *openssl.SSL) openssl.SSLTLSExtErr {
		log.Println("Using SNI to switch hostname:", ssl.GetServername())
		ctx := try(openssl.NewCtx())
		cert := try(MakeCert(ssl.GetServername()))
		ctx.UseCertificate(cert)
		ctx.UsePrivateKey(state.CAKey)
		ssl.SetSSLCtx(ctx)
		return openssl.SSLTLSExtErrOK
	})
	conn := try(openssl.Server(c, ctx))
	return conn
}
