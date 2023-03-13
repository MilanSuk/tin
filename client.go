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
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"

	"github.com/herumi/bls-eth-go-binary/bls"
)

type ClientAccount struct {
	key    bls.SecretKey
	pubKey BLSPubKey

	amount int64
	nonce  int64
}

func NewClientAccount(privKey *BLSPrivKey) (*ClientAccount, error) {
	var client ClientAccount

	if privKey != nil {
		err := privKey.Export(&client.key)
		if err != nil {
			return nil, fmt.Errorf("NewClientAccount() privKey export() failed: %w", err)
		}
	} else {
		client.key.SetByCSPRNG()
		privKey = NewBLSPrivKey(&client.key)
	}

	// get pubKey from key
	err := privKey.ExportPublicKey(&client.pubKey)
	if err != nil {
		return nil, fmt.Errorf("NewClientAccount() pubKey export() failed: %w", err)
	}

	return &client, nil
}

func _Client_GetRandomAccount(accounts []*ClientAccount, min_amount int64) int {

	var ret_i int
	var ret *ClientAccount

	for ret == nil || ret.amount < min_amount {
		ret_i = rand.Intn(len(accounts))
		ret = accounts[ret_i]
	}
	return ret_i
}

func Client_WriteInt(file *os.File, value int64) error {

	var t [8]byte
	binary.LittleEndian.PutUint64(t[:], uint64(value))
	_, err := file.Write(t[:])
	if err != nil {
		return fmt.Errorf("Client_WriteFileBlock() write1() failed: %w", err)
	}
	return nil

}

func Client_getOrGenerateGenesis(genesisPath string, amount *int64, privKey *BLSPrivKey) error {

	if OsFileExists(genesisPath) {
		//opens
		data, err := os.ReadFile(genesisPath)
		if err != nil {
			return fmt.Errorf("Client_getOrGenerateGenesis() ReadFile() failed: %w", err)
		}

		//reads
		pos := 0
		*amount = int64(binary.LittleEndian.Uint64(data[pos : pos+8]))
		pos += 8

		copy(privKey.arr[:], data[pos:pos+len(privKey.arr)])

		fmt.Printf("Reading genesis file(%s)\n", genesisPath)
	} else {

		// generates
		*amount = 100000000
		cl, err := NewClientAccount(nil)
		if err != nil {
			return fmt.Errorf("Client_getOrGenerateGenesis() NewClientAccount() failed: %w", err)
		}
		*privKey = *NewBLSPrivKey(&cl.key)
		cl = nil

		// writes
		f, err := os.Create(genesisPath)
		if err != nil {
			return fmt.Errorf("Client_getOrGenerateGenesis() Create() failed: %w", err)
		}
		defer f.Close()

		err = Client_WriteInt(f, *amount)
		if err != nil {
			return fmt.Errorf("Client_getOrGenerateGenesis() Client_WriteInt() failed: %w", err)
		}
		_, err = f.Write(privKey.arr[:])
		if err != nil {
			return fmt.Errorf("Client_getOrGenerateGenesis() Write() failed: %w", err)
		}

		fmt.Printf("Genesis file(%s) written\n", genesisPath)
	}

	return nil
}

func Client_generateTxnsFile(NUMBER_TXNS int, genesis_amount int64, genesis_privKey *BLSPrivKey, txnsPath string) error {
	//  void Client_generateTxnsFile(const UBIG NUMBER_TXNS, const UBIG genesis_amount, BLSPrivKey *genesis_privKey, const OsText txnsPath)

	f, err := os.Create(txnsPath)
	if err != nil {
		return fmt.Errorf("Client_generateTxnsFile() Create() failed: %w", err)
	}
	defer f.Close()

	// adds genesis
	genesis, err := NewClientAccount(genesis_privKey)
	if err != nil {
		return fmt.Errorf("Client_generateTxnsFile() NewClientAccount() failed: %w", err)
	}

	var accounts []*ClientAccount
	accounts = append(accounts, genesis)
	genesis.amount = genesis_amount

	// generates txns
	var txnBuff TBuffer
	st := OsTime()
	for i := 0; i < NUMBER_TXNS; i++ {

		var txn TxnRaw
		var src, dst *ClientAccount
		var amount int64
		if i%20 == 0 {
			amount = 1000

			// new account(from genesisAccount . newOne)
			src_i := 0
			//dst_i := len(accounts)

			src = accounts[src_i]
			dst, err = NewClientAccount(nil)
			if err != nil {
				return fmt.Errorf("Client_generateTxnsFile() NewClientAccount() failed: %w", err)
			}
			accounts = append(accounts, dst)

			txn.InitTxnRawLong(int64(src_i), src.nonce, amount, 0, &dst.pubKey)
		} else {
			amount = 1
			src_i := _Client_GetRandomAccount(accounts, amount)
			dst_i := _Client_GetRandomAccount(accounts, 0)

			src = accounts[src_i]
			dst = accounts[dst_i]

			txn.InitTxnRawShort(int64(src_i), src.nonce, amount, 0, int64(dst_i))
		}

		src.amount -= amount
		dst.amount += amount
		src.nonce++

		err := txn.ExportBuffer(&src.key, &txnBuff)
		if err != nil {
			return fmt.Errorf("Client_generateTxnsFile() ExportBuffer() failed: %w", err)
		}

		err = Client_WriteInt(f, int64(txnBuff.size))
		if err != nil {
			return fmt.Errorf("Client_generateTxnsFile() Client_WriteInt() failed: %w", err)
		}
		_, err = f.Write(txnBuff.data[:txnBuff.size])
		if err != nil {
			return fmt.Errorf("Client_generateTxnsFile() Write() failed: %w", err)
		}

		if i%500 == 0 {
			prc := float64(i) / float64(NUMBER_TXNS)
			dt := OsTime() - st
			fmt.Printf("%.1f%% remaining time %.1fmin\n", prc*100, ((dt/prc)-dt)/60)
		}
	}

	fmt.Printf("\nDone in time(%.1fsec). Txns(%d) generated into file(%s). Number of accounts(%d)\n", OsTime()-st, NUMBER_TXNS, txnsPath, len(accounts))

	return nil
}

func Client_sendTxns(node *Node, txnsPath string) (int, error) {

	data, err := os.ReadFile(txnsPath)
	if err != nil {
		return -1, fmt.Errorf("Client_sendTxns() ReadFile() failed: %w", err)
	}

	var num_added = 0
	pos := 0
	for pos < len(data) {
		bytes := int(binary.LittleEndian.Uint64(data[pos : pos+8]))
		pos += 8

		node.net.txnsPool.Add(data[pos : pos+bytes])
		pos += bytes

		num_added++

	}

	return num_added, nil
}

func Client_sendBlocks(node *Node, blocksPath string) (int, error) {

	data, err := os.ReadFile(blocksPath)
	if err != nil {
		return -1, fmt.Errorf("Client_sendBlocks() ReadFile() failed: %w", err)
	}

	var num_added = 0
	pos := 0
	for pos < len(data) {
		bytes := int(binary.LittleEndian.Uint64(data[pos : pos+8]))
		pos += 8

		node.net.blocksPool.Add(data[pos : pos+bytes])
		pos += bytes

		num_added++

	}

	return num_added, nil
}
