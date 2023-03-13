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
	"sync/atomic"
	"time"

	"github.com/herumi/bls-eth-go-binary/bls"
)

func BlockVerMT_GetStartEnd(thread_i int, num_txns int) (int, int) {

	n := OsMax(1, int(OsRoundUp(float64(num_txns)/float64(BlockVerMT_NUM_AGG_SIGNITURES))))
	return (thread_i * n), OsMin(num_txns, (thread_i*n)+n)
}

func _BlockVerMT_VerifyInner(st int, en int, aggSign *bls.Sign, block *BlockRaw, num_done *atomic.Uint32, out_ok *bool) {

	*out_ok = true
	if en > st {
		*out_ok = aggSign.AggregateVerifyNoCheck(block.pubKeys[st:en], block.hashes[st*32:en*32])
	}
	num_done.Add(1)
}

func BlockVerMT_Verify(aggSign []bls.Sign, block *BlockRaw) error {

	var num_done atomic.Uint32
	var oks [BlockVerMT_NUM_AGG_SIGNITURES]bool

	// runs
	for i := 0; i < BlockVerMT_NUM_AGG_SIGNITURES; i++ {
		st, en := BlockVerMT_GetStartEnd(i, block.NumTxns())
		go _BlockVerMT_VerifyInner(st, en, &aggSign[i], block, &num_done, &oks[i])
	}

	//waits
	for num_done.Load() < BlockVerMT_NUM_AGG_SIGNITURES {
		time.Sleep(1 * time.Millisecond)
	}

	//checks
	for i := 0; i < BlockVerMT_NUM_AGG_SIGNITURES; i++ {
		if !oks[i] {
			return errors.New("AggregateVerifyNoCheck() failed")
		}
	}

	return nil
}

func BlockVerMT_Sign(aggSign []bls.Sign, block *BlockRaw) {
	for i := 0; i < len(aggSign); i++ {
		st, en := BlockVerMT_GetStartEnd(i, block.NumTxns())
		if en > st {
			aggSign[i].Aggregate(block.signs[st:en])
		} else {
			aggSign[i] = bls.Sign{}
		}
	}
}
