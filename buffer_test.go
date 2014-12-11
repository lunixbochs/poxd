package main

import (
	"fmt"
	"log"
	"testing"
	"time"
)

func TestBuffer(t *testing.T) {
	log.Println("started")
	b := &Buffer{}
	b.Write([]byte("test 1"))
	b.Write([]byte("test 2"))

	done := make(chan int)
	go func(r *Reader) {
		for {
			buf := make([]byte, 1024)
			_, err := r.Read(buf, 1)
			if err != nil {
				break
			}
			log.Println("printing", string(buf))
		}
		log.Println("closed")
		done <- 1
	}(b.Reader())

	time.Sleep(1 * time.Second)
	for i := 0; i < 100; i++ {
		b.Write([]byte(fmt.Sprintf("(test %d)", i)))
	}
	b.Close()
	<-done
}
