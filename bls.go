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

func InitBLS() error {

	err := bls.Init(bls.BLS12_381)
	if err != nil {
		return err
	}

	bls.SetETHmode(bls.EthModeLatest) // use the latest eth2.0 spec

	return nil
}

type BLSPrivKey struct {
	arr [32]byte
}

type BLSPubKey struct {
	arr [48]byte
}

type BLSSign struct {
	arr [96]byte
}

func NewBLSPubKey(pubKey *bls.PublicKey) *BLSPubKey {
	var self BLSPubKey
	self.arr = [48]byte(pubKey.Serialize())

	return &self
}

func (pubKey *BLSPubKey) Export(exp *bls.PublicKey) error {
	err := exp.Deserialize(pubKey.arr[:])
	if err != nil {
		return err
	}
	return nil
}

func (a *BLSPubKey) Cmp(b *BLSPubKey) bool {
	return a.arr == b.arr
}

func NewBLSSign(sign *bls.Sign) *BLSSign {
	var self BLSSign
	self.arr = [96]byte(sign.Serialize())
	return &self
}

func (sign *BLSSign) Export(exp *bls.Sign) error {

	err := exp.Deserialize(sign.arr[:])
	if err != nil {
		return err
	}
	return nil
}

func NewBLSPrivKey(privKey *bls.SecretKey) *BLSPrivKey {
	var self BLSPrivKey
	self.arr = [32]byte(privKey.Serialize())
	return &self
}

func (privKey *BLSPrivKey) Export(exp *bls.SecretKey) error {
	err := exp.Deserialize(privKey.arr[:])
	if err != nil {
		return err
	}
	return nil
}

func (privKey *BLSPrivKey) ExportPublicKey(exp *BLSPubKey) error {

	var key bls.SecretKey
	err := privKey.Export(&key)
	if err != nil {
		return fmt.Errorf("GetPublicKey() export() failed: %w", err)
	}

	*exp = *NewBLSPubKey(key.GetPublicKey())
	return nil
}
