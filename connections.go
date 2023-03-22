/*
Copyright 2023 Milan Suk

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/gorilla/websocket"
)

type Connections struct {
	clients []*websocket.Conn
}

func NewConnections() *Connections {
	var self Connections
	return &self
}

func (cons *Connections) Destroy() {
	for _, c := range cons.clients {
		c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		c.Close()
	}
}

func (cons *Connections) Add(addr string, port int, path string, ssl_on bool) error {

	var ssl_proto string
	if ssl_on {
		ssl_proto = "wss"
	} else {
		ssl_proto = "ws"
	}

	u := url.URL{Scheme: ssl_proto, Host: addr + ":" + strconv.Itoa(port), Path: "/" + path} //wss = for SSL

	var c *websocket.Conn
	var err error
	if ssl_on {
		d := websocket.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		c, _, err = d.Dial(u.String(), nil)
	} else {
		c, _, err = websocket.DefaultDialer.Dial(u.String(), nil) //without SSL
	}

	if err != nil {
		return fmt.Errorf("NewClient(): Failed to connect to %s with error: %w", u.String(), err)
	}

	cons.clients = append(cons.clients, c)

	fmt.Printf("Client connected to %s\n", u.String())
	return nil
}

func (cons *Connections) Send(msg []byte) error {

	for _, c := range cons.clients {
		err := c.WriteMessage(websocket.BinaryMessage, msg)
		if err != nil {
			return fmt.Errorf("SendWrite(): WriteMessage() failed: %w", err)
		}
	}
	return nil
}

func (cons *Connections) SendTxn(txn []byte) error {

	if len(txn) == 0 {
		return errors.New("SendTxn() is empty")
	}

	var msg []byte
	msg = append(msg, MSG_TXN)
	msg = append(msg, txn...)

	return cons.Send(msg)
}

func (cons *Connections) SendBlock(block []byte) error {

	if len(block) == 0 {
		return errors.New("SendBlock() is empty")
	}

	var msg []byte
	msg = append(msg, MSG_BLOCK)
	msg = append(msg, block...)

	return cons.Send(msg)
}
