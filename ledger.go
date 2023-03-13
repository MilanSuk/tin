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
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type Ledger struct {
	accounts *Accounts

	dbPath string

	db             *sql.DB
	insertTxn      *sql.Stmt
	numRowsTxn     *sql.Stmt
	selectTxnBlock *sql.Stmt
}

func NewLedger(dbPath string) (*Ledger, error) {
	var self Ledger
	self.dbPath = dbPath

	var err error
	self.db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		self.Destroy()
		return nil, fmt.Errorf("NewDb() Open() failed: %w", err)
	}

	_, err = self.db.Exec("CREATE TABLE Txns(account_id INTEGER, amount INTEGER, nonce INTEGER, pre_rowid INTEGER);")
	if err != nil {
		self.Destroy()
		return nil, fmt.Errorf("NewLedger() Exec1() failed: %w", err)
	}

	_, err = self.db.Exec("CREATE INDEX IF NOT EXISTS TxnsIndex_account_id on Txns (account_id);")
	if err != nil {
		self.Destroy()
		return nil, fmt.Errorf("NewLedger() Exec2() failed: %w", err)
	}

	//_, err = self.db.Exec("CREATE INDEX IF NOT EXISTS TxnsIndex_time on Txns (time);")
	//if err != nil {
	//	return nil, fmt.Errorf("NewLedger() Exec1() failed: %w", err)
	//}

	self.insertTxn, err = self.db.Prepare("INSERT INTO Txns(account_id, amount, nonce, pre_rowid) VALUES(?,?,?,?);")
	if err != nil {
		self.Destroy()
		return nil, fmt.Errorf("NewLedger() insertTxn stmt failed: %w", err)
	}

	self.selectTxnBlock, err = self.db.Prepare("SELECT account_id, amount, nonce, pre_rowid, MAX(_rowid_) FROM Txns WHERE account_id >= ? AND account_id < ? AND _rowid_ <= ? GROUP BY account_id;")
	if err != nil {
		self.Destroy()
		return nil, fmt.Errorf("NewLedger() selectTxnBlock stmt failed: %w", err)
	}

	self.numRowsTxn, err = self.db.Prepare("SELECT COUNT(*) FROM Txns;")
	if err != nil {
		self.Destroy()
		return nil, fmt.Errorf("NewLedger() numRowsTxn stmt failed: %w", err)
	}

	self.accounts, err = NewAccounts(self.db)
	if err != nil {
		self.Destroy()
		return nil, fmt.Errorf("NewLedger() failed: %w", err)
	}

	return &self, nil
}

func (ledger *Ledger) Destroy() {
	if ledger.accounts != nil {
		ledger.accounts.Destroy()
	}

	if ledger.insertTxn != nil {
		ledger.insertTxn.Close()
	}
	if ledger.numRowsTxn != nil {
		ledger.numRowsTxn.Close()
	}
	if ledger.selectTxnBlock != nil {
		ledger.selectTxnBlock.Close()
	}

	if ledger.db != nil {
		ledger.db.Close()
	}
}

func (ledger *Ledger) BatchStart() error {
	_, err := ledger.db.Exec("BEGIN")
	if err != nil {
		return fmt.Errorf("BatchStart() failed: %w", err)
	}
	return nil
}

func (ledger *Ledger) BatchCommit() error {
	_, err := ledger.db.Exec("COMMIT")
	if err != nil {
		return fmt.Errorf("BatchCommit() failed: %w", err)
	}
	return nil
}

func (ledger *Ledger) BatchRollback() error {
	_, err := ledger.db.Exec("ROLLBACK")
	if err != nil {
		return fmt.Errorf("BatchRollback() failed: %w", err)
	}
	return nil
}

func (ledger *Ledger) AddTxn(account_id int64, amount int64, nonce int64, last_rowid int64) (int64, error) {

	res, err := ledger.insertTxn.Exec(account_id, amount, nonce, last_rowid)
	if err != nil {
		return -1, fmt.Errorf("AddTxn() Exec() failed: %w", err)
	}

	row, err := res.LastInsertId()
	if err != nil {
		return -1, fmt.Errorf("AddTxn() LastInsertId() failed: %w", err)
	}

	return row, nil
}

// max_rowid = "time"
func (ledger *Ledger) SelectTxnBlock(account_id_start int64, max_rowid int64) (int64, error) {
	rows, err := ledger.selectTxnBlock.Query(account_id_start, account_id_start+1024, max_rowid)
	if err != nil {
		return -1, fmt.Errorf("SelectTxnBlock() failed: %w", err)
	}
	defer rows.Close()

	sum := int64(0)
	for rows.Next() {
		var account_id int64
		var amount int64
		var nonce int64
		var pre_rowid int64
		var rowid int64

		err = rows.Scan(&account_id, &amount, &nonce, &pre_rowid, &rowid)
		if err != nil {
			return -1, fmt.Errorf("SelectTxnBlock() Scan() failed: %w", err)
		}

		sum += amount
	}

	return sum, nil
}

func (ledger *Ledger) GetMaxTxnRow() (int64, error) {

	rows, err := ledger.numRowsTxn.Query()
	if err != nil {
		return -1, fmt.Errorf("GetMaxTxnRow() failed: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return -1, fmt.Errorf("GetMaxTxnRow() rows.Next() failed: %w", err)
	}

	var numRows int64
	err = rows.Scan(&numRows)
	if err != nil {
		return -1, fmt.Errorf("NewAccounts() Scan() failed: %w", err)
	}

	return numRows, nil
}
