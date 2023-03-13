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
	"errors"
	"fmt"
)

type Account struct {
	pubKey BLSPubKey

	amount  int64
	nonce   int64
	txn_row int64 // Txns table has 'pre_rowid' which can be use like a time
}

type Accounts struct {
	accounts    []*Account
	pubKeyIndex map[[48]byte]int

	insertAccount  *sql.Stmt
	selectAccounts *sql.Stmt
	selectTxns     *sql.Stmt
}

func NewAccounts(db *sql.DB) (*Accounts, error) {
	var self Accounts

	self.pubKeyIndex = make(map[[48]byte]int)

	_, err := db.Exec("CREATE TABLE Accounts(pub_key BLOB);")
	if err != nil {
		self.Destroy()
		return nil, fmt.Errorf("NewAccounts() Exec() failed: %w", err)
	}

	self.insertAccount, err = db.Prepare("INSERT INTO Accounts(pub_key) VALUES(?);")
	if err != nil {
		self.Destroy()
		return nil, fmt.Errorf("NewAccounts() insertAccount stmt failed: %w", err)
	}

	self.selectAccounts, err = db.Prepare("SELECT pub_key FROM Accounts ORDER BY _rowid_;")
	if err != nil {
		self.Destroy()
		return nil, fmt.Errorf("NewAccounts() selectAccounts stmt failed: %w", err)
	}

	self.selectTxns, err = db.Prepare("SELECT amount, nonce, MAX(_rowid_) FROM Txns GROUP BY account_id;")
	if err != nil {
		self.Destroy()
		return nil, fmt.Errorf("NewAccounts() selectTxns stmt failed: %w", err)
	}

	// reads from pubKeys
	{
		rows, err := self.selectAccounts.Query()
		if err != nil {
			self.Destroy()
			return nil, fmt.Errorf("NewAccounts() selectAccounts.Query() failed: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var data []byte
			err := rows.Scan(&data)
			if err != nil {
				self.Destroy()
				return nil, fmt.Errorf("NewAccounts() selectAccounts.Scan() failed: %w", err)
			}

			var pubKey BLSPubKey
			if len(data) != len(pubKey.arr) {
				self.Destroy()
				return nil, fmt.Errorf("NewAccounts() Wrong Public key size(%d) failed: %w", len(data), err)
			}

			pubKey.arr = [48]byte(data)
			_, err = self._Add(&pubKey, false)
			if err != nil {
				self.Destroy()
				return nil, fmt.Errorf("NewAccounts() _Add() failed: %w", err)
			}
		}
	}

	// reads from Account attributes
	{
		rows, err := self.selectTxns.Query()
		if err != nil {
			return nil, fmt.Errorf("NewAccounts() selectTxns.Query() failed: %w", err)
		}
		defer rows.Close()

		i := 0
		for rows.Next() {
			if i >= len(self.accounts) {
				return nil, fmt.Errorf("NewAccounts() selectTxns out of accounts len: %w", err)
			}

			acc := self.accounts[i]
			err := rows.Scan(&acc.amount, &acc.nonce, &acc.txn_row)
			if err != nil {
				return nil, fmt.Errorf("NewAccounts() selectTxns.Scan() failed: %w", err)
			}
		}
	}

	return &self, nil
}

func (accs *Accounts) Destroy() {
	if accs.insertAccount != nil {
		accs.insertAccount.Close()
	}
	if accs.selectAccounts != nil {
		accs.selectAccounts.Close()
	}
	if accs.selectTxns != nil {
		accs.selectTxns.Close()
	}
}

func (accs *Accounts) Get(i int) (*Account, error) {

	if i < 0 || i >= len(accs.accounts) {
		return nil, errors.New("out of index")
	}

	ret := accs.accounts[i]
	if ret == nil {
		return nil, errors.New("Account is null")
	}
	return ret, nil
}

func (accs *Accounts) Find(pubKey *BLSPubKey) (int, error) {

	pos, ok := accs.pubKeyIndex[pubKey.arr]

	if !ok {
		return -1, errors.New("PubKey not found")
	}

	return pos, nil
}

func (accs *Accounts) _Add(pubKey *BLSPubKey, insertIntoDb bool) (int, error) {

	i := len(accs.accounts)

	accs.accounts = append(accs.accounts, &Account{pubKey: *pubKey})

	accs.pubKeyIndex[pubKey.arr] = i

	// adds to SQLite
	if insertIntoDb {
		_, err := accs.insertAccount.Exec(pubKey.arr[:])
		if err != nil {
			return -1, fmt.Errorf("_Add() Exec() failed: %w", err)
		}
	}

	return i, nil
}

func (accs *Accounts) Add(pubKey *BLSPubKey) (int, error) {

	i, err := accs.Find(pubKey)
	if err != nil {
		i, err = accs._Add(pubKey, true)
		if err != nil {
			return -1, err
		}
	}

	return i, nil
}

func (accs *Accounts) Rollback() {
	// resize down accounts ...
	// re-read amount,nonce,txn_row from Txns table ...
	// remove indexes bigger than max_accounts ...
}

func (accs *Accounts) SumAmounts() int64 {

	sum := int64(0)
	for _, it := range accs.accounts {
		sum += it.amount
	}
	return sum
}
