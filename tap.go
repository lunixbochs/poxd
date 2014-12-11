package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"

	"code.google.com/p/go-uuid/uuid"
	"github.com/lunixbochs/redigo/redis"
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

type Request struct {
	Id       string `json:"id"`
	Method   string `json:"method"`
	Uri      string `json:"uri"`
	Host     string `json:"host"`
	Request  string `json:"request"`
	Response string `json:"response"`
}

type QertMessage struct {
	Type    string   `json:"type"`
	Request *Request `json:"request"`
}

func LogRequest(req *http.Request, resp *http.Response) error {
	red := try(redis.Dial("tcp", state.Redis))
	var rawReq, rawResp bytes.Buffer
	req.Write(&rawReq)
	resp.Write(&rawResp)

	log.Println(req.Header)

	r := &Request{
		Id:       uuid.New(),
		Method:   req.Method,
		Uri:      req.RequestURI,
		Host:     req.Host,
		Request:  rawReq.String(),
		Response: rawResp.String(),
	}
	data := try(json.Marshal(r))
	red.Do("LPUSH", "qert-history", data)
	data = try(json.Marshal(&QertMessage{"request", r}))
	red.Do("PUBLISH", "qert", data)
	return nil
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
			if req != nil {
				err = LogRequest(req, nil)
				if err != nil {
					log.Println(err)
				}
			}
			break
		}
		resp, err := http.ReadResponse(bufio.NewReader(remoteReader), req)
		if resp != nil {
			err2 := LogRequest(req, resp)
			if err2 != nil {
				log.Println(err)
			}
		}
		log.Println(resp)
		if err != nil {
			log.Println(err)
			break
		}
	}
}
