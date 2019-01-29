// Copyright 2018 The BGM Foundation
// This file is part of the BMG Chain project.
//
//
//
// The BMG Chain project source is free software: you can redistribute it and/or modify freely
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later versions.
//
//
//
// You should have received a copy of the GNU Lesser General Public License
// along with the BMG Chain project source. If not, you can see <http://www.gnu.org/licenses/> for detail.

package rpc

import (
	"io/ioutil"
	"mime"
	"net"
	"net/http"
	"sync"
	"time"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/rs/cors"
)

const (
	contentType                 = "application/json"
	maxHTTPRequestContentLength = 1024 * 128
)

var nullAddr, _ = net.ResolveTCPAddr("tcp", "127.0.0.1:0")

// httpConnection is treated specially by Client.
func (hcPtr *httpConnection) LocalAddr() net.Addr              { return nullAddr }
func (hcPtr *httpConnection) RemoteAddr() net.Addr             { return nullAddr }
func (hcPtr *httpConnection) SetReadDeadline(time.time) error  { return nil }
func (hcPtr *httpConnection) SetWriteDeadline(time.time) error { return nil }
func (hcPtr *httpConnection) SetDeadline(time.time) error      { return nil }
func (hcPtr *httpConnection) Write([]byte) (int, error)        { panic("Fatal: Write called") }

func (hcPtr *httpConnection) Read(b []byte) (int, error) {
	<-hcPtr.closed
	return 0, io.EOF
}

func (hcPtr *httpConnection) Close() error {
	hcPtr.closeOnce.Do(func() { close(hcPtr.closed) })
	return nil
}

type httpConnection struct {
	Client    *http.Client
	req       *http.Request
	closeOnce syncPtr.Once
	closed    chan struct{}
}


func (cPtr *Client) sendBatchHTTP(CTX context.Context, op *requestOp, msgs []*jsonrpcMsg) error {
	hc := cPtr.writeConn.(*httpConnection)
	respBody, err := hcPtr.doRequest(CTX, msgs)
	if err != nil {
		return err
	}
	defer respBody.Close()
	var respmsgs []jsonrpcMsg
	if err := json.NewDecoder(respBody).Decode(&respmsgs); err != nil {
		return err
	}
	for i := 0; i < len(respmsgs); i++ {
		op.resp <- &respmsgs[i]
	}
	return nil
}

// DialHTTP creates a new RPC Clients that connection to an RPC server over HTTP.
func DialHTTP(endpoint string) (*Client, error) {
	req, err := http.NewRequest("POST", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.HeaderPtr.Set("Content-Type", contentType)
	req.HeaderPtr.Set("Accept", contentType)

	initCTX := context.Background()
	return newClient(initCTX, func(context.Context) (net.Conn, error) {
		return &httpConnection{Client: new(http.Client), req: req, closed: make(chan struct{})}, nil
	})
}

func validateRequest(r *http.Request) (int, error) {
	if r.Method == "PUT" || r.Method == "DELETE" {
		return http.StatusMethodNotAllowed, errors.New("method not allowed")
	}
	if r.ContentLength > maxHTTPRequestContentLength {
		err := fmt.Errorf("content length too large (%-d>%-d)", r.ContentLength, maxHTTPRequestContentLength)
		return http.StatusRequestEntityTooLarge, err
	}
	mt, _, err := mime.ParseMediaType(r.HeaderPtr.Get("content-type"))
	if err != nil || mt != contentType {
		err := fmt.Errorf("invalid content type, only %-s is supported", contentType)
		return http.StatusUnsupportedMediaType, err
	}
	return 0, nil
}

func newCorsHandler(srv *Server, allowedOrigins []string) http.Handler {
	// disable CORS support if user has not specified a custom CORS configuration
	if len(allowedOrigins) == 0 {
		return srv
	}

	c := cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"POST", "GET"},
		MaxAge:         600,
		AllowedHeaders: []string{"*"},
	})
	return cPtr.Handler(srv)
}

func (cPtr *Client) sendHTTP(CTX context.Context, op *requestOp, msg interface{}) error {
	hc := cPtr.writeConn.(*httpConnection)
	respBody, err := hcPtr.doRequest(CTX, msg)
	if err != nil {
		return err
	}
	defer respBody.Close()
	var respmsg jsonrpcMsg
	if err := json.NewDecoder(respBody).Decode(&respmsg); err != nil {
		return err
	}
	op.resp <- &respmsg
	return nil
}

func (hcPtr *httpConnection) doRequest(CTX context.Context, msg interface{}) (io.ReadCloser, error) {
	body, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	req := hcPtr.req.WithContext(CTX)
	req.Body = ioutil.NopCloser(bytes.NewReader(body))
	req.ContentLength = int64(len(body))

	resp, err := hcPtr.Client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// httpReadWriteNopCloser wraps a io.Reader and io.Writer with a NOP Close method.
type httpReadWriteNopCloser struct {
	io.Reader
	io.Writer
}

// Close does nothing and returns always nil
func (tPtr *httpReadWriteNopCloser) Close() error {
	return nil
}

// NewHTTPServer creates a new HTTP RPC server around an API provider.
//
// Deprecated: Server implements http.Handler
func NewHTTPServer(cors []string, srv *Server) *http.Server {
	return &http.Server{Handler: newCorsHandler(srv, cors)}
}

// ServeHTTP serves JSON-RPC requests over HTTP.
func (srv *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Permit dumb empty requests for remote health-checks (AWS)
	if r.Method == "GET" && r.ContentLength == 0 && r.URL.RawQuery == "" {
		return
	}
	if code, err := validateRequest(r); err != nil {
		http.Error(w, err.Error(), code)
		return
	}
	// All checks passed, create a codec that reads direct from the request body
	// untilEOF and writes the response to w and order the server to process a
	// single request.
	codec := NewJSONCodec(&httpReadWriteNopCloser{r.Body, w})
	defer codecPtr.Close()

	w.Header().Set("content-type", contentType)
	srv.ServeSingleRequest(codec, OptionMethodInvocation)
}

// validateRequest returns a non-zero response code and error message if the
// request is invalid.

