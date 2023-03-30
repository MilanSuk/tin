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

	"github.com/herumi/bls-eth-go-binary/bls"
)

const (
	TxnRaw_LONG  = 0
	TxnRaw_SHORT = 1
)

const TxnRaw_ADJUST_SRC_ID = 1000000
const TxnRaw_ADJUST_DST_ID = 1000000
const TxnRaw_ADJUST_AMOUNT = 1000000

const BlockVerMT_NUM_AGG_SIGNITURES = 8

type TxnRaw struct {
	src_id    int64
	src_nonce int64
	amount    int64
	fee       int64 // compression? ...

	//union? ...
	dst_pubKey BLSPubKey
	dst_id     int64

	dst_type uint8
}

func (txn *TxnRaw) InitTxnRaw(src_id int64, src_nonce int64, amount int64, fee int64, dst_type uint8) {

	txn.src_id = src_id + TxnRaw_ADJUST_SRC_ID
	txn.src_nonce = src_nonce
	txn.amount = amount + TxnRaw_ADJUST_AMOUNT
	txn.fee = fee
	txn.dst_type = dst_type
}

func (txn *TxnRaw) InitTxnRawLong(src_id int64, src_nonce int64, amount int64, fee int64, dst_pubKey *BLSPubKey) {
	txn.InitTxnRaw(src_id, src_nonce, amount, fee, TxnRaw_LONG)
	txn.dst_pubKey = *dst_pubKey
}

func (txn *TxnRaw) InitTxnRawShort(src_id int64, src_nonce int64, amount int64, fee int64, dst_id int64) {
	txn.InitTxnRaw(src_id, src_nonce, amount, fee, TxnRaw_SHORT)
	txn.dst_id = dst_id + TxnRaw_ADJUST_DST_ID
}

func (txn *TxnRaw) InitTxnFromBuffer(buff *TBuffer, readPubKey bool, needSign bool) ([]byte, *bls.PublicKey, *bls.Sign, error) {
	var pk bls.PublicKey
	if readPubKey {
		pubKey := BLSPubKey{}
		err := buff.ReadSBlob(pubKey.arr[:], int64(len(pubKey.arr)))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("InitTxnFromBuffer() failed: %w", err)
		}

		err = pubKey.Export(&pk)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("InitTxnFromBuffer() PubKey export failed: %w", err)
		}
	}

	pos_backup := buff.pos

	var err error
	txn.src_id, err = buff.ReadNumber()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("InitTxnFromBuffer() failed: %w", err)
	}
	txn.src_nonce, err = buff.ReadNumber()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("InitTxnFromBuffer() failed: %w", err)
	}
	txn.amount, err = buff.ReadNumber()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("InitTxnFromBuffer() failed: %w", err)
	}
	txn.fee, err = buff.ReadNumber()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("InitTxnFromBuffer() failed: %w", err)
	}
	txn.dst_type, err = buff.ReadByte()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("InitTxnFromBuffer() failed: %w", err)
	}

	txn.src_id -= TxnRaw_ADJUST_SRC_ID
	txn.amount -= TxnRaw_ADJUST_AMOUNT

	if txn.dst_type == 0 {
		err := buff.ReadSBlob(txn.dst_pubKey.arr[:], int64(len(txn.dst_pubKey.arr)))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("InitTxnFromBuffer() failed: %w", err)
		}
	} else {
		txn.dst_id, err = buff.ReadNumber()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("InitTxnFromBuffer() failed: %w", err)
		}

		txn.dst_id -= TxnRaw_ADJUST_DST_ID
	}

	ret := buff.data[pos_backup:buff.pos]
	var sig bls.Sign
	if needSign {
		var s BLSSign
		err := buff.ReadSBlob(s.arr[:], int64(len(s.arr)))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("InitTxnFromBuffer() failed: %w", err)
		}

		err = s.Export(&sig)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("InitTxnFromBuffer() failed: %w", err)
		}
	}

	return ret, &pk, &sig, nil
}

func (txn *TxnRaw) ExportBuffer(pubKey *BLSPubKey, key *bls.SecretKey, buff *TBuffer) error {

	buff.Clear()
	if pubKey != nil {
		buff.WriteSBlob(pubKey.arr[:])
	}

	buff.WriteNumber(txn.src_id)
	buff.WriteNumber(txn.src_nonce)
	buff.WriteNumber(txn.amount)
	buff.WriteNumber(txn.fee)

	buff.WriteUint8(txn.dst_type)

	if txn.dst_type == 0 {
		buff.WriteSBlob(txn.dst_pubKey.arr[:])
	} else {
		buff.WriteNumber(txn.dst_id)
	}

	{
		h, err := buff.sha256(len(pubKey.arr))
		if err != nil {
			return fmt.Errorf("ExportBuffer() failed: %w", err)
		}

		s := NewBLSSign(key.SignByte(h)) // SLOW
		buff.WriteSBlob(s.arr[:])
	}

	return nil
}
