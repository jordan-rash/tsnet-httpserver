package main

import (
	httpserver "github.com/wasmcloud/interfaces/httpserver/tinygo"
	msgpack "github.com/wasmcloud/tinygo-msgpack"
)

func encodeRequest(req httpserver.HttpRequest) ([]byte, error) {
	var sizer msgpack.Sizer
	size_enc := &sizer
	req.MEncode(size_enc)
	buf := make([]byte, sizer.Len())
	encoder := msgpack.NewEncoder(buf)
	enc := &encoder
	req.MEncode(enc)

	return buf, encoder.CheckError()
}

// resp = core.HttpResponse
func decodeResponse(resp []byte) (*httpserver.HttpResponse, error) {
	d := msgpack.NewDecoder(resp)
	msg, err := httpserver.MDecodeHttpResponse(&d)
	if err != nil {
		return nil, err
	}

	return &msg, nil
}
