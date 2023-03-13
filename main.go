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
	"os"
	"time"
)

func main() {

	InitBLS()

	//MinerTest()

	//file paths
	err := os.Mkdir("data", os.ModePerm)
	if err != nil {
		fmt.Printf("Mkdir() failed: %v", err)
		return
	}
	dbPathA := "data/dbA.sqlite"
	dbPathB := "data/dbB.sqlite"
	genesisPath := "data/genesis.bin"
	txnsPath := "data/txns.bin"
	blocksPath := "data/blocks.bin"

	const NUMBER_TXNS = 40000
	const NUMBER_TXNS_IN_BLOCK = 10000

	// inits Db
	var genesis_amount int64
	var genesis_privKey BLSPrivKey
	err = Client_getOrGenerateGenesis(genesisPath, &genesis_amount, &genesis_privKey)
	if err != nil {
		fmt.Printf("Client_getOrGenerateGenesis() failed: %v", err)
		return
	}
	var genesis_pubKey BLSPubKey
	genesis_privKey.ExportPublicKey(&genesis_pubKey)

	// generates txns into write them into file
	{
		if !OsFileExists(txnsPath) {
			err = Client_generateTxnsFile(NUMBER_TXNS, genesis_amount, &genesis_privKey, txnsPath)
			if err != nil {
				fmt.Printf("Client_generateTxnsFile() failed: %v", err)
				return
			}
		}
	}

	// recvs txns and build blocks
	{
		OsFileRemove(dbPathA)
		OsFileRemove(blocksPath)
		node, err := NewNode(dbPathA, NUMBER_TXNS_IN_BLOCK, genesis_amount, &genesis_pubKey, blocksPath) //blocksPath=write blocks into file
		if err != nil {
			fmt.Printf("NewNode() failed: %v\n", err)
			return
		}

		n, err := Client_sendTxns(node, txnsPath)
		if err != nil {
			fmt.Printf("Client_sendTxns() failed: %v", err)
			return
		}

		for node.stat.sum_txns < n {
			time.Sleep(1 * time.Millisecond)
		}

		node.Destroy()
	}

	// recvs blocks and verify them
	{
		OsFileRemove(dbPathB)
		node, err := NewNode(dbPathB, NUMBER_TXNS_IN_BLOCK, genesis_amount, &genesis_pubKey, "")
		if err != nil {
			fmt.Printf("NewNode() failed: %v\n", err)
			return
		}

		n, err := Client_sendBlocks(node, blocksPath)
		if err != nil {
			fmt.Printf("Client_sendBlocks() failed: %v", err)
			return
		}

		for node.stat.num_blocks < n {
			time.Sleep(1 * time.Millisecond)
		}

		node.Destroy()
	}
}
