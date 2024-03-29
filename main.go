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
	"log"
	"os"
	"runtime"
	"strconv"
	"time"
)

func main() {

	InitBLS()

	//MinerTest()

	//file paths
	os.Mkdir("data", os.ModePerm)
	dbPathA := "data/dbA.sqlite"
	dbPathB := "data/dbB.sqlite"
	genesisPath := "data/genesis.bin"
	txnsPath := "data/txns_"
	blocksPath := "data/blocks.bin"

	const NUMBER_TXNS = 40000
	const NUMBER_TXNS_IN_BLOCK = 10000

	// inits Db
	var genesis_amount int64
	var genesis_privKey BLSPrivKey
	err := Client_getOrGenerateGenesis(genesisPath, &genesis_amount, &genesis_privKey)
	if err != nil {
		log.Printf("Client_getOrGenerateGenesis() failed: %v\n", err)
		return
	}
	var genesis_pubKey BLSPubKey
	genesis_privKey.ExportPublicKey(&genesis_pubKey)

	// generates txns into write them into file
	{
		if !OsFileExists(txnsPath + "0") {
			err = Client_generateTxnsFile(NUMBER_TXNS, NUMBER_TXNS_IN_BLOCK, genesis_amount, &genesis_privKey, txnsPath)
			if err != nil {
				log.Printf("Client_generateTxnsFile() failed: %v\n", err)
				return
			}
		}
	}

	const PORT = 4879
	// recvs txns and build blocks
	{
		OsFileRemove(dbPathA)
		OsFileRemove(blocksPath)
		node, err := NewNode(false, PORT, dbPathA, NUMBER_TXNS_IN_BLOCK, genesis_amount, &genesis_pubKey, blocksPath) //blocksPath=write blocks into file
		if err != nil {
			log.Printf("NewNode() failed: %v\n", err)
			return
		}

		time.Sleep(100 * time.Millisecond)
		var conns []*Connections
		for i := 0; i < runtime.NumCPU(); i++ {
			conns = append(conns, NewConnections())
			err = conns[i].Add("localhost", PORT, "data", false)
			if err != nil {
				log.Printf("onnections.Add() failed: %v\n", err)
				return
			}
		}

		n, err := Client_sendTxns(conns[0], txnsPath+"0")
		if err != nil {
			log.Printf("Client_sendTxns() failed: %v\n", err)
			return
		}
		nn := n
		for node.stat.sum_txns < nn {
			time.Sleep(1 * time.Millisecond)
		}

		for bi := 0; bi < (NUMBER_TXNS/NUMBER_TXNS_IN_BLOCK)-1; bi++ {
			time.Sleep(1 * time.Second) //? ...
			n, err := Client_sendTxnsMT(conns, txnsPath+strconv.Itoa(bi+1))
			if err != nil {
				log.Printf("Client_sendTxns() failed: %v\n", err)
				return
			}
			nn += n
			for node.stat.sum_txns < nn {
				time.Sleep(1 * time.Millisecond)
			}
		}

		for _, c := range conns {
			c.Destroy()
		}
		node.Destroy()
	}

	// recvs blocks and verify them
	{
		OsFileRemove(dbPathB)
		node, err := NewNode(false, PORT, dbPathB, NUMBER_TXNS_IN_BLOCK, genesis_amount, &genesis_pubKey, "")
		if err != nil {
			log.Printf("NewNode() failed: %v\n", err)
			return
		}

		conns := NewConnections()
		err = conns.Add("localhost", PORT, "data", false)
		if err != nil {
			log.Printf("Connections.Add() failed: %v\n", err)
			return
		}

		n, err := Client_sendBlocks(conns, blocksPath)
		if err != nil {
			log.Printf("Client_sendBlocks() failed: %v\n", err)
			return
		}

		for node.stat.num_blocks < n {
			time.Sleep(1 * time.Millisecond)
		}

		conns.Destroy()
		node.Destroy()
	}
}
