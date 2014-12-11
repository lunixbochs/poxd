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
	pos, buf int
}

func (b *Buffer) poke() {
	for _, r := range b.readers {
		if len(r.Ready) == 0 {
			r.Ready <- 1
		}
	}
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

func (r *Reader) Read(p []byte, timeout int) (int, error) {
	max := len(p)
	pos := 0
	// block if we don't have any messages ready
	if r.buf >= len(r.messages) {
		if r.Closed {
			return 0, errors.New("end of file")
		}
		select {
		case <-r.Ready:
		case <-time.After(time.Duration(timeout) * time.Second):
			return 0, errors.New("buffer read timed out")
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
