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
	"errors"
	"sync"
)

type Pool struct {
	lock sync.Mutex

	items [][]byte
}

func NewPool() *Pool {
	var self Pool
	return &self
}

func (pool *Pool) Num() int {

	pool.lock.Lock()
	defer pool.lock.Unlock()
	return len(pool.items)
}

func (pool *Pool) Add(item []byte) {

	pool.lock.Lock()
	defer pool.lock.Unlock()

	pool.items = append(pool.items, item)
}

func (pool *Pool) Get() ([]byte, error) {

	pool.lock.Lock()
	defer pool.lock.Unlock()

	if len(pool.items) == 0 {
		return nil, errors.New("Pool is empty")
	}

	ret := pool.items[0]
	pool.items = pool.items[1:] //remove
	return ret, nil
}
