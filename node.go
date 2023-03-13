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
)

const BlocksPool_ITEM = 1024 * 1024

type NodeStat struct {
	start_time float64

	sum_time   float64
	sum_txns   int
	sum_bytes  int
	num_blocks int
}

func (stat *NodeStat) Start() {
	stat.start_time = OsTime()
}

func (stat *NodeStat) End(bytes int, num_txns int) {
	stat.sum_time += OsTime() - stat.start_time

	stat.num_blocks++
	stat.sum_txns += num_txns
	stat.sum_bytes += bytes
}

func (stat *NodeStat) AvgTxnInBlock() int {
	return OsTrn(stat.num_blocks > 0, stat.sum_txns/stat.num_blocks, 0)
}

func (stat *NodeStat) AvgBytesInBlock() int {
	return OsTrn(stat.num_blocks > 0, stat.sum_bytes/stat.num_blocks, 0)
}
func (stat *NodeStat) AvgTimeBlock() float64 {
	return OsTrnFloat(stat.num_blocks > 0, stat.sum_time/float64(stat.num_blocks), 0)
}

func (stat *NodeStat) Print(ledger *Ledger) {

	fmt.Printf("Num Accounts: %d\n", len(ledger.accounts.accounts))
	fmt.Printf("Num blocks: %d\n", stat.num_blocks)

	fmt.Printf("Avg txn in block written: %d\n", stat.AvgTxnInBlock())
	fmt.Printf("Avg bytes in block written: %.1f%% of 1MB\n", float64(stat.AvgBytesInBlock())/BlocksPool_ITEM*100)
	fmt.Printf("Avg block time: %.2fsec\n", stat.AvgTimeBlock())
	fmt.Printf("Db file size: %.1fMB\n", float64(OsFileBytes(ledger.dbPath))/1024.0/1024.0)
	fmt.Printf("Avg db bytes/txn: %.dB\n", OsTrn(stat.sum_txns > 0, int(OsFileBytes(ledger.dbPath))/stat.sum_txns, 0))

	fmt.Println("------")

}

type Node struct {
	net    *Net
	ledger *Ledger

	blockRaw BlockRaw
	block    TBuffer
	txn      TBuffer

	stat NodeStat

	blocksFile           *os.File
	NUMBER_TXNS_IN_BLOCK int

	thread OsThread
}

func NewNode(dbPath string, NUMBER_TXNS_IN_BLOCK int, genesis_amount int64, genesis_pubKey *BLSPubKey, blocksPath string) (*Node, error) {
	var node Node
	var err error

	node.ledger, err = NewLedger(dbPath)
	if err != nil {
		return nil, fmt.Errorf("NewNode() NewLedger failed: %w", err)
	}

	node.net, err = NewNet()
	if err != nil {
		return nil, fmt.Errorf("NewNode() NewNet failed: %w", err)
	}

	node.NUMBER_TXNS_IN_BLOCK = NUMBER_TXNS_IN_BLOCK

	// adds genesis account
	ac_id, err := node.ledger.accounts.Add(genesis_pubKey)
	if err != nil {
		return nil, fmt.Errorf("NewNode() Add() genesis account failed: %w", err)
	}

	ac, err := node.ledger.accounts.Get(ac_id)
	if err != nil {
		return nil, fmt.Errorf("NewNode() Get() genesis account failed: %w", err)
	}
	ac.amount = genesis_amount
	ac.nonce = 0

	if len(blocksPath) > 0 {
		node.blocksFile, err = os.OpenFile(blocksPath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			return nil, fmt.Errorf("NewNode() Creating blocks file failed: %w", err)
		}
	}

	go node.Loop()

	return &node, nil
}

func (node *Node) Destroy() {

	node.thread.Wait()

	err := node.net.Destroy()
	if err != nil {
		fmt.Printf("Node.Destroy() failed: %v", err)
	}
	node.ledger.Destroy()

	err = node.blocksFile.Close()
	if err != nil {
		fmt.Printf("Node.Close() failed: %v", err)
	}
}

func (node *Node) CreateBlock() error {

	if node.net.txnsPool.Num() >= node.NUMBER_TXNS_IN_BLOCK { // or timeout ...

		node.stat.Start()
		node.ledger.BatchStart()

		// add txns into new block
		var absErr error
		for i := 0; i < node.NUMBER_TXNS_IN_BLOCK; i++ {

			txn, err := node.net.txnsPool.Get()
			if err != nil {
				absErr = fmt.Errorf("CreateBlock() Get() failed: %w", err)
				break
			}
			node.txn.WriteSBlob(txn)

			isFull, err := node.blockRaw.AddTxn(&node.txn, BlocksPool_ITEM, &node.block, node.ledger)
			if err != nil {
				absErr = fmt.Errorf("CreateBlock() AddTxn() failed: %w", err)
				break
			}
			if isFull {
				break
			}
			//... node.net.txnsPool.Add(node.txn.data[:node.txn.size]) // returns txn back to pool
		}

		if absErr == nil {
			node.ledger.BatchCommit()
		} else {
			node.ledger.BatchRollback()
			node.ledger.accounts.Rollback() //? ...
			return absErr
		}

		// finish block
		node.blockRaw.Finish(&node.block)
		// BlocksPool_addBlock(node.net.blocksPool, node.block)
		if node.blocksFile != nil {

			err := Client_WriteInt(node.blocksFile, node.block.size)
			if err != nil {
				return fmt.Errorf("CreateBlock() Client_WriteInt() failed: %w", err)
			}
			_, err = node.blocksFile.Write(node.block.data[:node.block.size])
			if err != nil {
				return fmt.Errorf("CreateBlock() Write() failed: %w", err)
			}
		}

		node.stat.End(int(node.block.size), node.blockRaw.NumTxns())
		node.stat.Print(node.ledger)
		node.blockRaw.ResetAndPrepare(&node.block)
	}

	return nil
}

func (node *Node) VerifyBlock() error {
	block, err := node.net.blocksPool.Get()
	if err != nil {
		//	return fmt.Errorf("VerifyBlock() Get() failed: %w", err)
		return nil
	}
	node.block.Clear()
	node.block.WriteSBlob(block)

	node.stat.Start()

	err = node.blockRaw.CheckAndWrite(&node.block, node.ledger)
	if err != nil {
		return fmt.Errorf("VerifyBlock() CheckAndWrite() failed: %w", err)
	}

	node.stat.End(int(node.block.size), node.blockRaw.NumTxns())
	node.stat.Print(node.ledger)

	return nil
}

func (node *Node) Loop() {

	defer node.thread.End()

	node.blockRaw.ResetAndPrepare(&node.block)

	for node.thread.Is() {
		err := node.CreateBlock()
		if err != nil {
			fmt.Printf("Loop() CreateBlock() failed: %v\n", err)
		}

		err = node.VerifyBlock()
		if err != nil {
			fmt.Printf("Loop() VerifyBlock() failed: %v\n", err)
		}
	}
}
