package main

import (
	"bufio"
	"io"
	"log"
	"net/http"
	"sync"
)

type Copy struct {
	To   io.Writer
	From io.Reader
}

type Tap struct {
	io.Writer
	Messages chan []byte
}

func NewTap(w io.Writer) *Tap {
	return &Tap{Writer: w, Messages: make(chan []byte, 50)}
}

func (t *Tap) Write(p []byte) (int, error) {
	if len(p) > 0 {
		c := make([]byte, len(p))
		copy(c, p)
		t.Messages <- c
	} else {
		close(t.Messages)
	}
	return t.Writer.Write(p)
}

func MultiCopy(copies ...Copy) {
	var wg sync.WaitGroup
	for _, c := range copies {
		go func(c Copy) {
			io.Copy(c.To, c.From)
		}(c)
	}
	wg.Add(len(copies))
	wg.Wait()
}

func Pipe(client, remote io.ReadWriter) {
	MultiCopy(Copy{client, remote}, Copy{remote, client})
}

func LogHttp(client, remote io.ReadWriter) {
	clientTap := NewTap(client)
	remoteTap := NewTap(remote)
	go MultiCopy(Copy{clientTap, remote}, Copy{remoteTap, client})

	clientBuffer := &Buffer{}
	remoteBuffer := &Buffer{}
	go clientBuffer.Consume(remoteTap.Messages)
	go remoteBuffer.Consume(clientTap.Messages)

	clientReader := clientBuffer.Reader()
	clientReader.SetTimeout(5)
	remoteReader := remoteBuffer.Reader()
	remoteReader.SetTimeout(5)

	for {
		req, err := http.ReadRequest(bufio.NewReader(clientReader))
		log.Println(req)
		if err != nil {
			log.Println(err)
			break
		}
		resp, err := http.ReadResponse(bufio.NewReader(remoteReader), req)
		log.Println(resp)
		if err != nil {
			log.Println(err)
			break
		}
	}
}
