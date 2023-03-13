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
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/bits"
	"math/rand"
	"runtime"
	"sync/atomic"
	"time"
)

func DoubleHashH(b []byte) [32]byte {
	first := sha256.Sum256(b)
	return sha256.Sum256(first[:])
}

type Miner struct {
	num_done      atomic.Uint32
	num_threads   int
	resultChannel chan BlockHeader

	inf_num_hashes atomic.Uint64
	start_time     time.Time
}

func (miner *Miner) _MinerWorker(header BlockHeader) {

	buff := header.Serialize()

	for miner.num_done.Load() == 0 {

		N := 1000
		for i := 0; i < N; i++ {
			hash := DoubleHashH(buff)

			if bits.LeadingZeros64(binary.LittleEndian.Uint64(hash[:])) >= int(header.bits) {
				miner.resultChannel <- header

				miner.num_done.Add(1)
				miner.inf_num_hashes.Add(uint64(i))
				break
			}
			header.nonce++
			binary.LittleEndian.PutUint32(buff[len(buff)-4:], header.nonce)
		}
		miner.inf_num_hashes.Add(uint64(N))
	}

}

func NewMiner(header BlockHeader, num_threads int, resultChannel chan BlockHeader) *Miner {

	if num_threads < 1 {
		num_threads = runtime.NumCPU()
	}

	var miner Miner
	miner.num_threads = num_threads
	miner.resultChannel = resultChannel
	miner.start_time = time.Now()

	for i := 0; i < num_threads; i++ {
		header.nonce = rand.Uint32()
		go miner._MinerWorker(header)
	}

	return &miner
}

func (miner *Miner) StopMiner() {
	miner.num_done.Add(1)
}

func MinerTest() {

	resultChannel := make(chan BlockHeader)

	header := BlockHeader{}
	header.bits = 26
	miner := NewMiner(header, -1, resultChannel)

	//time.Sleep(100 * time.Millisecond)
	//miner.StopMiner()

	ticker := time.NewTicker(1 * time.Second)

out:
	for {
		select {
		case resultHeader := <-resultChannel:
			fmt.Println(resultHeader)
			break out

		case <-ticker.C:
			t := time.Since(miner.start_time)
			fmt.Printf("Speed: %v hashes/sec\n", int64(float64(miner.inf_num_hashes.Load())/t.Seconds()))
		}
	}

	t := time.Since(miner.start_time)
	fmt.Printf("Time: %v\n", t)
}
