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
	"fmt"

	"github.com/herumi/bls-eth-go-binary/bls"
)

type BlockRaw struct {
	pubKeys []bls.PublicKey
	hashes  []byte
	signs   []bls.Sign
}

func (block *BlockRaw) NumTxns() int {
	return len(block.hashes) / 32
}

func (block *BlockRaw) Clear() {
	block.pubKeys = block.pubKeys[:0]
	block.hashes = block.hashes[:0]
	block.signs = block.signs[:0]
}

func (block *BlockRaw) ResetAndPrepare(buff *TBuffer) {

	block.Clear()
	buff.Clear()

	// Block always starts with Signiture
	var aggSigns [BlockVerMT_NUM_AGG_SIGNITURES]BLSSign

	for i := 0; i < len(aggSigns); i++ {
		buff.WriteSBlob(aggSigns[i].arr[:])
	}
}

func (block *BlockRaw) _Add(pubKey *bls.PublicKey, msg []byte, sign *bls.Sign) error {

	if pubKey != nil {
		block.pubKeys = append(block.pubKeys, *pubKey)
	}
	if msg != nil {
		h, err := TBuffer_sha256(msg)
		if err != nil {
			return fmt.Errorf("BlockRaw._Add() sha256 failed: %w", err)
		}

		block.hashes = append(block.hashes, h...)
	}
	if sign != nil {
		block.signs = append(block.signs, *sign)
	}
	return nil
}

func _BlockRaw_AddTxnIntoAccount(txn *TxnRaw, ledger *Ledger) (*Account, error) {

	// get src
	srcAcc, err := ledger.accounts.Get(int(txn.src_id))
	if err != nil {
		return nil, fmt.Errorf("_BlockRaw_AddTxnIntoAccount() get src_id failed: %w", err)
	}

	// check src
	if srcAcc.nonce != txn.src_nonce {
		return nil, errors.New("wrong nonce")
	}
	if srcAcc.amount < txn.amount {
		return nil, errors.New("wron amount")
	}

	// get dst
	var dst_i int
	if txn.dst_type == TxnRaw_SHORT {
		dst_i = int(txn.dst_id)
	} else if txn.dst_type == TxnRaw_LONG {
		dst_i, err = ledger.accounts.Add(&txn.dst_pubKey)
		if err != nil {
			return nil, fmt.Errorf("_BlockRaw_AddTxnIntoAccount() add dst_pubKey failed: %w", err)
		}
	}
	dstAcc, err := ledger.accounts.Get(dst_i)
	if err != nil {
		return nil, fmt.Errorf("_BlockRaw_AddTxnIntoAccount() get dst_id failed: %w", err)
	}

	//move
	srcAcc.nonce++
	srcAcc.amount -= txn.amount
	dstAcc.amount += txn.amount
	// fee ...

	srcAcc.txn_row, err = ledger.AddTxn(txn.src_id, srcAcc.amount, srcAcc.nonce, srcAcc.txn_row)
	if err != nil {
		return nil, fmt.Errorf("_BlockRaw_AddTxnIntoAccount() AddTxn1() failed: %w", err)
	}
	dstAcc.txn_row, err = ledger.AddTxn(int64(dst_i), dstAcc.amount, dstAcc.nonce, dstAcc.txn_row)
	if err != nil {
		return nil, fmt.Errorf("_BlockRaw_AddTxnIntoAccount() AddTxn2() failed: %w", err)
	}
	return srcAcc, nil
}

func (block *BlockRaw) AddTxn(txnBuff *TBuffer, max_block_size int, blockBuff *TBuffer, ledger *Ledger) (bool, error) {

	var txn TxnRaw
	msg, sign, err := txn.InitTxnFromBuffer(txnBuff, true)
	if err != nil {
		return false, fmt.Errorf("AddTxn().InitTxnFromBuffer() failed: %w", err)
	}

	if int(blockBuff.size)+len(msg) > max_block_size {
		return true, nil
	}

	account, err := ledger.accounts.Get(int(txn.src_id))
	if err != nil {
		return false, fmt.Errorf("AddTxn() get src_id failed: %w", err)
	}

	var pk bls.PublicKey
	err = account.pubKey.Export(&pk)
	if err != nil {
		return false, fmt.Errorf("AddTxn() src export pubKey failed: %w", err)
	}

	h, err := TBuffer_sha256(msg)
	if err != nil {
		return false, fmt.Errorf("AddTxn() sha256() failed: %w", err)
	}

	if !sign.VerifyByte(&pk, h) { // SLOW
		return false, errors.New("AddTxn() Signiture is invalid")
	}

	_, err = _BlockRaw_AddTxnIntoAccount(&txn, ledger)
	if err != nil {
		return false, fmt.Errorf("AddTxn() _BlockRaw_AddTxnIntoAccount() failed: %w", err)
	}

	// add
	err = block._Add(nil, msg, sign)
	if err != nil {
		return false, fmt.Errorf("AddTxn() _Add() failed: %w", err)
	}
	blockBuff.WriteSBlob(msg)

	return false, nil
}

func (block *BlockRaw) Finish(blockBuff *TBuffer) error {
	var aggSigns [BlockVerMT_NUM_AGG_SIGNITURES]bls.Sign
	BlockVerMT_Sign(aggSigns[:], block)

	var signs [BlockVerMT_NUM_AGG_SIGNITURES]BLSSign
	for i := 0; i < len(signs); i++ {
		signs[i] = *NewBLSSign(&aggSigns[i])
	}

	// checks and writes at the buffer start
	if blockBuff.size < int64(len(signs[0].arr)*BlockVerMT_NUM_AGG_SIGNITURES) {
		return errors.New("AddTxn() Buffer is too short for agg signitures")
	}
	for i := 0; i < len(signs); i++ {
		copy(blockBuff.data[i*len(signs[0].arr):], signs[i].arr[:])
	}

	return nil
}

func (block *BlockRaw) CheckAndWrite(blockBuff *TBuffer, ledger *Ledger) error {

	blockBuff.pos = 0

	var aggSigns [BlockVerMT_NUM_AGG_SIGNITURES]bls.Sign
	for i := 0; i < len(aggSigns); i++ {

		var sg BLSSign
		err := blockBuff.ReadSBlob(sg.arr[:], int64(len(sg.arr)))
		if err != nil {
			return fmt.Errorf("CheckAndWrite() Buffer read failed: %w", err)
		}

		err = sg.Export(&aggSigns[i])
		if err != nil {
			return fmt.Errorf("CheckAndWrite() aggsign export failed: %w", err)
		}
	}

	ledger.BatchStart()
	block.Clear()

	var absError error
	for blockBuff.pos < blockBuff.size {
		var txn TxnRaw
		msg, _, err := txn.InitTxnFromBuffer(blockBuff, false)
		if err != nil {
			absError = fmt.Errorf("CheckAndWrite() InitTxnFromBuffer() failed: %w", err)
			break
		}

		srcAcc, err := _BlockRaw_AddTxnIntoAccount(&txn, ledger)
		if err != nil {
			absError = fmt.Errorf("CheckAndWrite() _BlockRaw_AddTxnIntoAccount() failed: %w", err)
			break
		}

		var pubKey bls.PublicKey
		err = srcAcc.pubKey.Export(&pubKey)
		if err != nil {
			absError = fmt.Errorf("CheckAndWrite() pubKey Export() failed: %w", err)
			break
		}

		err = block._Add(&pubKey, msg, nil)
		if err != nil {
			absError = fmt.Errorf("CheckAndWrite() _Add() failed: %w", err)
			break
		}
	}

	if absError == nil {
		err := BlockVerMT_Verify(aggSigns[:], block) // SLOWER(multi-threaded)
		//err := blsAggregateVerifyNoCheck(&aggSign, self.pubKeys, self.hashes, sizeof(OsHsh32), self.num_txns)
		if err != nil {
			absError = fmt.Errorf("CheckAndWrite() Verify() failed: %w", err)
		}
	}

	if absError == nil {
		ledger.BatchCommit()
	} else {
		ledger.BatchRollback()
		ledger.accounts.Rollback()
	}

	return absError
}
