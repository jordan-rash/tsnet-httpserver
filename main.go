package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"

	provider "github.com/wasmCloud/provider-sdk-go"
	core "github.com/wasmcloud/interfaces/core/tinygo"
	httpserver "github.com/wasmcloud/interfaces/httpserver/tinygo"

	"tailscale.com/tsnet"
)

var (
	p  *provider.WasmcloudProvider
	s  *tsnet.Server
	ln net.Listener

	ErrNotImplemented   error = errors.New("operation not implemented")
	ErrInvalidOperation error = errors.New("operation not valid")
)

func main() {
	var err error

	p, err = provider.New(
		"wasmcloud:httpserver",
		provider.WithNewLinkFunc(handleNewLink),
		provider.WithDelLinkFunc(handleDelLink),
		provider.WithHealthCheckMsg(healthCheckMsg),
	)
	if err != nil {
		panic(err)
	}

	err = p.Start()
	if err != nil {
		panic(err)
	}
}

func healthCheckMsg() string {
	return ""
}

func handleNewLink(linkdef core.LinkDefinition) error {
	var err error
	port := linkdef.Values["port"]
	hostname := linkdef.Values["hostname"]
	ts_authkey := linkdef.Values["ts_authkey"]

	if port == "" || hostname == "" || ts_authkey == "" {
		return errors.New("invalid link settings")
	}

	s = &tsnet.Server{
		Hostname: hostname,
		AuthKey:  ts_authkey,
	}

	ln, err = s.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal(err)
	}

	return http.Serve(ln,
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				headers := httpserver.HeaderMap{}
				for k, v := range r.Header {
					headers[k] = v
				}
				headers["wasmcloud_provider"] = httpserver.HeaderValues{"true"}

				body, err := ioutil.ReadAll(r.Body)
				if err != nil {
					w.WriteHeader(500)
					w.Write([]byte(err.Error()))
				}

				req := httpserver.HttpRequest{
					Method:      r.Method,
					Path:        r.URL.Path,
					QueryString: r.URL.RawQuery,
					Header:      headers,
					Body:        body,
				}

				encReq, err := encodeRequest(req)
				if err != nil {
					w.WriteHeader(500)
					w.Write([]byte(err.Error()))
					return
				}

				resp, err := p.ToActor(
					linkdef.ActorId,
					encReq,
					"HttpServer.HandleRequest",
				)
				if err != nil {
					w.WriteHeader(500)
					w.Write([]byte(err.Error()))
					return
				}

				respDecoded, err := decodeResponse(resp)
				if err != nil {
					w.WriteHeader(500)
					w.Write([]byte(err.Error()))
					return
				}

				addHeaders(w, headers)

				sURL := strings.Split(r.URL.Path, ".")
				end := sURL[len(sURL)-1]
				switch end {
				case "css":
					p.Logger.Info("Adding CSS header to " + r.URL.Path)
					w.Header().Set("Content-Type", "text/css")
					p.Logger.Info(fmt.Sprintf("%v", w.Header()))
				}

				w.WriteHeader(int(respDecoded.StatusCode))
				w.Write(respDecoded.Body)
			},
		),
	)
}

func handleDelLink(_ core.LinkDefinition) error {
	ln.Close()
	return s.Close()
}

func addHeaders(w http.ResponseWriter, headers httpserver.HeaderMap) {
	for k, v := range headers {
		for _, tV := range v {
			if w.Header().Get(k) == "" {
				w.Header().Set(k, tV)
			} else {
				w.Header().Add(k, tV)
			}
		}
	}
}
