package main

import (
	"errors"
	"time"
)

type Buffer struct {
	messages [][]byte
	readers  []*Reader
	Closed   bool
}

type Reader struct {
	*Buffer
	Ready    chan int
	Timeout  time.Duration
	pos, buf int
}

func (b *Buffer) poke() {
	for _, r := range b.readers {
		if len(r.Ready) == 0 {
			r.Ready <- 1
		}
	}
}

func (b *Buffer) Consume(c chan []byte) {
	for msg := range c {
		b.Write(msg)
	}
	b.Close()
}

func (b *Buffer) Write(p []byte) {
	b.messages = append(b.messages, p)
	b.poke()
}

func (b *Buffer) Close() {
	b.Closed = true
	b.poke()
}

func (b *Buffer) Reader() *Reader {
	r := &Reader{Buffer: b, Ready: make(chan int, 1)}
	b.readers = append(b.readers, r)
	return r
}

func (r *Reader) SetTimeout(timeout time.Duration) {
	r.Timeout = timeout
}

func (r *Reader) Read(p []byte) (int, error) {
	max := len(p)
	pos := 0
	// block if we don't have any messages ready
	if r.buf >= len(r.messages) {
		if r.Closed {
			return 0, errors.New("end of file")
		}
		if r.Timeout == 0 {
			<-r.Ready
		} else {
			select {
			case <-r.Ready:
			case <-time.After(r.Timeout * time.Second):
				return 0, errors.New("buffer read timed out")
			}
		}
	}
	// copy buffers into dest until we run out
	for pos < max && r.buf < len(r.messages) {
		buf := r.messages[r.buf]
		copy(p[pos:], buf[r.pos:])
		pos += len(buf[r.pos:])
		r.pos = 0
		r.buf++
	}
	return pos, nil
}
