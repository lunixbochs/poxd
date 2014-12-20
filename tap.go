package main

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"code.google.com/p/go-uuid/uuid"
	"github.com/lunixbochs/mgo/bson"
	"github.com/lunixbochs/redigo/redis"
)

type Copy struct {
	To   io.Writer
	From io.Reader
}

type Tap struct {
	io.Writer
	Messages chan []byte
	eof      chan int
	Closed   bool
}

func NewTap(w io.Writer) *Tap {
	return &Tap{
		Writer:   w,
		Messages: make(chan []byte, 50),
		eof:      make(chan int, 1),
	}
}

func (t *Tap) Write(p []byte) (int, error) {
	if len(p) > 0 {
		c := make([]byte, len(p))
		copy(c, p)
		t.Messages <- c
	} else {
		close(t.Messages)
		t.Closed = true
		t.eof <- 1
	}
	return t.Writer.Write(p)
}

func (t *Tap) Reader() *Reader {
	return &Reader{Tap: t}
}

type Reader struct {
	*Tap
	pending []byte
	pos     int
	Timeout time.Duration
}

func (r *Reader) SetTimeout(timeout time.Duration) {
	r.Timeout = timeout
}

func (r *Reader) Read(p []byte) (int, error) {
	max := len(p)
	pos := 0
	// block if we don't have a message ready
	if r.pending == nil || r.pos >= len(r.pending) {
		if r.Closed {
			return 0, errors.New("end of file")
		}
		select {
		case r.pending = <-r.Messages:
		case <-r.eof:
			return 0, errors.New("EOF")
		case <-time.After(r.Timeout):
			return 0, errors.New("buffer read timed out")
		}
	}
	// copy buffers into dest until we would block
loop:
	for pos < max && r.pending != nil {
		n := copy(p[pos:], r.pending[r.pos:])
		pos += n
		r.pos += n
		if r.pos >= len(r.pending) {
			r.pos = 0
			select {
			case r.pending = <-r.Messages:
			default:
				r.pending = nil
				break loop
			}
		}
	}
	return pos, nil
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

type Request struct {
	Id       string `bson:"id"`
	Method   string `bson:"method"`
	Uri      string `bson:"uri"`
	Host     string `bson:"host"`
	Request  string `bson:"request"`
	Response string `bson:"response"`
}

type QertMessage struct {
	Type    string   `bson:"type"`
	Request *Request `bson:"request"`
}

func LogRequest(req *http.Request, resp *http.Response) error {
	red := try(redis.Dial("tcp", state.Redis))
	var rawReq, rawResp bytes.Buffer
	req.Write(&rawReq)
	resp.Write(&rawResp)

	r := &Request{
		Id:       uuid.New(),
		Method:   req.Method,
		Uri:      req.RequestURI,
		Host:     req.Host,
		Request:  rawReq.String(),
		Response: rawResp.String(),
	}
	data := try(bson.Marshal(r))
	red.Do("LPUSH", "qert-history", data)
	data = try(bson.Marshal(&QertMessage{"request", r}))
	red.Do("PUBLISH", "qert", data)
	return nil
}

func LogHttp(client, remote io.ReadWriter) {
	clientTap := NewTap(client)
	remoteTap := NewTap(remote)
	go MultiCopy(Copy{clientTap, remote}, Copy{remoteTap, client})

	toClient := clientTap.Reader()
	toClient.SetTimeout(5 * time.Second)
	toRemote := remoteTap.Reader()
	toRemote.SetTimeout(5 * time.Second)

	for {
		req, err := http.ReadRequest(bufio.NewReader(toRemote))
		log.Println("req", req)
		if err != nil {
			log.Println(err)
			if req != nil {
				err = LogRequest(req, nil)
				if err != nil {
					log.Println(err)
				}
			}
			break
		}
		resp, err := http.ReadResponse(bufio.NewReader(toClient), req)
		if resp != nil {
			err2 := LogRequest(req, resp)
			if err2 != nil {
				log.Println(err)
			}
		}
		log.Println("resp", resp)
		if err != nil {
			log.Println(err)
			break
		}
	}
}
