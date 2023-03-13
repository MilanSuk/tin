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
	"time"
)

type BlockHeader struct {
	version uint8

	prevBlock  [32]byte
	merkleRoot [32]byte

	timestamp time.Time

	bits  uint32 // difficulty
	nonce uint32 //salt
}

func (h *BlockHeader) Serialize() []byte {

	buff := make([]byte, 128)
	pos := 0

	buff[pos] = h.version
	pos++

	copy(buff[pos:], h.prevBlock[:])
	pos += 32

	copy(buff[pos:], h.merkleRoot[:])
	pos += 32

	binary.LittleEndian.PutUint32(buff[pos:], uint32(h.timestamp.Unix()))
	pos += 4

	binary.LittleEndian.PutUint32(buff[pos:], h.bits)
	pos += 4

	binary.LittleEndian.PutUint32(buff[pos:], h.nonce)
	pos += 4

	return buff[:pos]
}

func (h *BlockHeader) Deserialize(buff []byte) {
	pos := 0

	h.version = buff[pos]
	pos++

	copy(h.prevBlock[:], buff[pos:pos+32])
	pos += 32

	copy(h.merkleRoot[:], buff[pos:pos+32])
	pos += 32

	h.timestamp = time.Unix(int64(binary.LittleEndian.Uint32(buff[pos:])), 0)
	pos += 4

	h.bits = binary.LittleEndian.Uint32(buff[pos:])
	pos += 4

	h.nonce = binary.LittleEndian.Uint32(buff[pos:])
	pos += 4
}
