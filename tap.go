package main

import (
	"io"
	"log"
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
	go func() {
		for v := range clientTap.Messages {
			log.Println(string(v))
		}
	}()
	go func() {
		for v := range remoteTap.Messages {
			log.Println(string(v))
		}
	}()
}
