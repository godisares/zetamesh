// Copyright 2020 ZetaMesh Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
	"github.com/zetamesh/zetamesh/constant"
	"github.com/zetamesh/zetamesh/message"
	"github.com/zetamesh/zetamesh/version"
)

// client is used to access with the remote gateway
type client struct {
	gateway string
	key     string
	tls     bool
}

// newClient returns a new client instance which can be used to interact
// with the gateway.
func newClient(gateway, key string, tls bool) *client {
	return &client{
		gateway: gateway,
		key:     key,
		tls:     tls,
	}
}

// OpenTunnel request the server to open tunnel between the two peers.
// The source and destionation virtual network address need to be provided.
func (c *client) OpenTunnel(src, dst string) error {
	req := message.OpenTunnelRequest{
		Version:     version.NewVersion().String(),
		Source:      src,
		Destination: dst,
	}
	res := message.OpenTunnelResponse{}
	err := c.post(constant.URIOpenTunnel, req, &res)
	if err != nil {
		return errors.WithMessage(err, "open tunnel failed")
	}

	return nil
}

func (c *client) do(method, api string, reader io.Reader, res interface{}) error {
	var prefix string
	if c.tls {
		prefix = fmt.Sprintf("https://%s", c.gateway)
	} else {
		prefix = fmt.Sprintf("http://%s", c.gateway)
	}
	url := prefix + api
	request, err := http.NewRequest(method, url, reader)
	if err != nil {
		return errors.WithStack(err)
	}
	defer request.Body.Close()

	// Set the request headers
	request.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return errors.WithStack(err)
	}

	type response struct {
		Code  message.StatusCode `json:"code"`
		Error string             `json:"error"`
		Data  interface{}        `json:"data"`
	}

	result := &response{Data: res}
	err = json.NewDecoder(resp.Body).Decode(result)
	if err != nil {
		return errors.WithMessagef(err, "invalid json response when request %s", api)
	}

	if result.Code != message.StatusCode_Success {
		return fmt.Errorf("%d: %s", result.Code, result.Error)
	}

	return nil
}

//nolint:unused
func (c *client) get(api string, res interface{}) error {
	return c.do(http.MethodGet, api, nil, res)
}

func (c *client) post(api string, req, res interface{}) error {
	buffer := &bytes.Buffer{}
	err := json.NewEncoder(buffer).Encode(req)
	if err != nil {
		return errors.WithStack(err)
	}

	return c.do(http.MethodPost, api, buffer, res)
}
