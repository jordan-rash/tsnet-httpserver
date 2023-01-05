package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"

	provider "github.com/wasmCloud/provider-sdk-go"
	core "github.com/wasmcloud/interfaces/core/tinygo"
	httpserver "github.com/wasmcloud/interfaces/httpserver/tinygo"

	"tailscale.com/tsnet"
)

var (
	p      *provider.WasmcloudProvider
	s      *tsnet.Server
	ln     net.Listener
	fln    *tsnet.FunnelListener
	funnel bool = false
	useTLS bool = false

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
	var clientCert tls.Certificate

	port := linkdef.Values["port"]
	hostname := linkdef.Values["hostname"]
	ts_authkey := linkdef.Values["ts_authkey"]
	tls_private_key := linkdef.Values["tls_private_key"]
	tls_cert := linkdef.Values["tls_cert"]
	if linkdef.Values["funnel"] != "" {
		funnel, err = strconv.ParseBool(linkdef.Values["funnel"])
		if err != nil {
			p.Logger.Error(err, "Failed to parse funnel input")
		}
	}

	if port == "" || hostname == "" || ts_authkey == "" {
		return errors.New("invalid link settings")
	}

	if tls_cert != "" && tls_private_key != "" && !funnel {
		func() {
			tlsCertDec, err := base64.StdEncoding.DecodeString(tls_cert)
			if err != nil {
				p.Logger.Error(err, "Failed to decode tls certificate")
				return
			}
			tlsPriKeyDec, err := base64.StdEncoding.DecodeString(tls_private_key)
			if err != nil {
				p.Logger.Error(err, "Failed to decode tls private key")
				return
			}
			clientCert, err = tls.X509KeyPair(tlsCertDec, tlsPriKeyDec)
			if err != nil {
				p.Logger.Error(err, "Failed to create x509 key pair")
				return
			}
			useTLS = true
		}()
	}

	s = &tsnet.Server{
		Hostname: hostname,
		AuthKey:  ts_authkey,
	}

	lc, err := s.LocalClient()
	if err != nil {
		log.Fatal(err)
	}

	if funnel {
		fln, err = s.ExposeHTTPS()
		if err != nil {
			log.Fatal(err)
		}

		lln, err := s.Listen("tcp", ":443")
		if err != nil {
			log.Fatal(err)
		}

		ln = tls.NewListener(lln, &tls.Config{
			GetCertificate: lc.GetCertificate,
		})
	} else {
		ln, err = s.Listen("tcp", ":"+port)
		if err != nil {
			log.Fatal(err)
		}
		if useTLS {
			ln = tls.NewListener(ln, &tls.Config{
				Certificates: []tls.Certificate{clientCert},
				RootCAs:      x509.NewCertPool(),
			})
		}
	}

	return http.Serve(ln,
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				who, err := lc.WhoIs(r.Context(), r.RemoteAddr)
				if err != nil {
					w.WriteHeader(500)
					w.Write([]byte(err.Error()))
					return
				}

				headers := httpserver.HeaderMap{}
				for k, v := range r.Header {
					headers[k] = v
				}
				headers["wasmcloud_provider"] = httpserver.HeaderValues{"true"}
				headers["X-Webauth-User"] = httpserver.HeaderValues{who.UserProfile.ID.String()}
				headers["X-Webauth-Name"] = httpserver.HeaderValues{who.UserProfile.DisplayName}
				headers["X-Webauth-Login"] = httpserver.HeaderValues{who.UserProfile.LoginName}
				headers["X-Webauth-Tailnet"] = httpserver.HeaderValues{who.Node.Name}
				headers["X-Webauth-Profile-Picture"] = httpserver.HeaderValues{who.UserProfile.ProfilePicURL}

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

				// This little hack fixes CSS incase the actor isn't
				// setting content-type headers correctly
				sURL := strings.Split(r.URL.Path, ".")
				end := sURL[len(sURL)-1]
				switch end {
				case "css":
					w.Header().Set("Content-Type", "text/css")
					p.Logger.Info(fmt.Sprintf("%v", w.Header()))
				case "js":
					w.Header().Set("Content-Type", "application/javascript")
					p.Logger.Info(fmt.Sprintf("%v", w.Header()))
				}

				w.WriteHeader(int(respDecoded.StatusCode))
				w.Write(respDecoded.Body)
			},
		),
	)
}

func handleDelLink(_ core.LinkDefinition) error {
	if funnel {
		fln.Close()
	}
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
