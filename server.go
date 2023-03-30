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
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
)

type Server struct {
	txnsPool   *PoolTxns
	blocksPool *PoolBlocks

	server         http.Server
	isServerClosed bool
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func NewNet(ssl_on bool, port int) (*Server, error) {
	var net Server

	net.txnsPool = NewPoolTxns()
	net.blocksPool = NewPoolBlocks()

	go net.Loop(ssl_on, port)

	return &net, nil
}

func (net *Server) Destroy() error {

	net.isServerClosed = true
	net.server.Close()
	return nil
}

const MSG_TXN = 0
const MSG_BLOCK = 1

func (net *Server) Loop(ssl_on bool, port int) error {

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		fmt.Printf("Client accepted %s\n", r.URL.Path)

		if r.URL.Path == "/" {
			http.ServeFile(w, r, "tin.html")
			return
		} else if r.URL.Path == "/data" {

			c, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				log.Printf("Error: RunHub() failed: %v\n", err)
				return
			}
			defer c.Close()

			for {

				mt, message, err := c.ReadMessage()

				if mt == websocket.CloseMessage || websocket.IsCloseError(err) || websocket.IsUnexpectedCloseError(err) {
					return
				}

				if err != nil {
					log.Printf("Error: ReadMessage() failed: %v\n", err)
					return
				}

				if mt != websocket.BinaryMessage {
					log.Printf("Error: ReadMessage() is not binary: %v\n", err)
					return
				}

				if len(message) < 2 {
					log.Printf("Error: ReadMessage() is too small: %v\n", err)
					return
				}

				//var ans []byte
				if message[0] == MSG_TXN {
					message = message[1:]

					var txn TxnRaw
					msg, pubKey, sign, err := txn.InitTxnFromBuffer(NewTBuffer(message), true, true)
					if err != nil {
						log.Printf("Error: InitTxnFromBuffer() failed: %v", err)
						return
					}
					h, err := TBuffer_sha256(msg)
					if err != nil {
						log.Printf("Error: sha256() failed: %v", err)
						return
					}

					if !sign.VerifyByte(pubKey, h) { //SLOW ...
						log.Printf("Error: Signiture is invalid")
						return
					}

					net.txnsPool.Add(message) //including pubKey

				} else if message[0] == MSG_BLOCK {
					net.blocksPool.Add(message[1:])
				}

				/*err = c.WriteMessage(mt, ans)
				if err != nil {
					log.Println("Error: WriteMessage() failed: %w", err)
					return
				}*/
			}
		}
	})

	net.server = http.Server{Addr: ":" + strconv.Itoa(port), Handler: mux}

	var err error
	if ssl_on {
		err = net.server.ListenAndServeTLS("ssl/cert.pem", "ssl/key.pem")
	} else {
		err = net.server.ListenAndServe()
	}

	//show error only if loop is not closed naturaly
	if err != nil && !net.isServerClosed {
		log.Printf("Net.Loop() failed: %v\n", err)
		return err
	}

	//for net.thread.Is() {
	//	time.Sleep(10 * time.Millisecond)
	//}

	return nil
}
