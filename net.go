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

import "time"

type Net struct {
	txnsPool   *Pool
	blocksPool *Pool

	thread OsThread
}

func NewNet() (*Net, error) {
	var net Net

	net.txnsPool = NewPool()
	net.blocksPool = NewPool()

	go net.Loop()

	return &net, nil
}

func (net *Net) Destroy() error {

	net.thread.Wait()

	return nil
}

func (net *Net) Loop() {

	defer net.thread.End()

	for net.thread.Is() {
		// recv ...
		// TxnsPool_addTxn(TxnsPool *self, const OsBuff txn);
		// BlocksPool_addBlock(BlocksPool *self, const OsBuff block);

		time.Sleep(1 * time.Millisecond)
	}
}
