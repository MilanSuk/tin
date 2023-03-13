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
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
)

type TBuffer struct {
	data []byte
	size int64
	pos  int64
}

func TNewBuffer(src []byte) *TBuffer {
	var buff TBuffer

	if src == nil {
		buff.data = make([]byte, 4) //pre alloc
	} else {
		buff.data = src
		buff.size = int64(len(src))
	}

	return &buff
}

func NewTBufferCopy(src *TBuffer) *TBuffer {
	var buff TBuffer

	buff.pos = src.pos
	buff.size = src.size
	buff.data = make([]byte, len(src.data))
	copy(buff.data, src.data)
	return &buff
}

func (buff *TBuffer) Clear() {
	buff.pos = 0
	buff.size = 0
}

func TBuffer_sha256(data []byte) ([]byte, error) {
	h := sha256.New()
	_, err := h.Write(data)
	if err != nil {
		return nil, fmt.Errorf("TBuffer_sha256() failed: %w", err)
	}
	return h.Sum(nil), nil
}

func (buff *TBuffer) sha256() ([]byte, error) {

	return TBuffer_sha256(buff.data[:buff.size])
}

func (buff *TBuffer) Resize(bytes int64) {
	if bytes > int64(len(buff.data)) {
		data := make([]byte, bytes*2)
		copy(data, buff.data)
		buff.data = data
	}
}

func (buff *TBuffer) WriteUint8(value byte) {

	buff.Resize(buff.size + 1)
	buff.data[buff.size] = value
	buff.size++
}

/*
func (buff *TBuffer) WriteInt(value int64) {

		buff.Resize(buff.size + 8)
		binary.LittleEndian.PutUint64(buff.data[buff.size:], uint64(value))
		buff.size += 8
	}

func (buff *TBuffer) WriteFloat(value float64) {

		buff.Resize(buff.size + 8)
		binary.LittleEndian.PutUint64(buff.data[buff.size:], math.Float64bits(value))
		buff.size += 8
	}

func (buff *TBuffer) WriteString(value string) {

		len := int64(len(value))
		buff.WriteInt(len)

		buff.Resize(buff.size + len)
		copy(buff.data[buff.size:], value)
		buff.size += len
	}

func (buff *TBuffer) WriteBlob(value []byte) {

		len := int64(len(value))
		buff.WriteInt(len)

		buff.Resize(buff.size + len)
		copy(buff.data[buff.size:], value)
		buff.size += len
	}
*/
func (buff *TBuffer) WriteSBlob(value []byte) {

	len := int64(len(value))
	buff.Resize(buff.size + len)
	copy(buff.data[buff.size:], value)
	buff.size += len
}

func (buff *TBuffer) ReadByte() (byte, error) {

	if buff.pos+1 <= buff.size {
		value := buff.data[buff.pos]
		buff.pos++
		return value, nil
	}

	return 0, errors.New("ReadByte() is is of buffer")
}

func (buff *TBuffer) ReadNumber() (int64, error) {

	mask := uint8(buff.data[buff.pos])
	buff.pos++

	var t [8]byte

	for i := 0; i < 8; i++ {
		if mask&(1<<i) != 0 {
			if buff.pos+1 <= buff.size {
				t[i] = buff.data[buff.pos]
				buff.pos++
			} else {
				return 0, errors.New("ReadNumber() is is of buffer")
			}
		}
	}

	return int64(binary.LittleEndian.Uint64(t[:])), nil
}

/*func (buff *TBuffer) ReadInt() (int64, error) {

	if buff.pos+8 <= buff.size {
		value := int64(binary.LittleEndian.Uint64(buff.data[buff.pos:]))
		buff.pos += 8
		return value, nil
	}

	return 0, errors.New("ReadInt() is is of buffer")
}
func (buff *TBuffer) ReadFloat() (float64, error) {

	if buff.pos+8 <= buff.size {
		value := math.Float64frombits(binary.LittleEndian.Uint64(buff.data[buff.pos:]))
		buff.pos += 8
		return value, nil
	}

	return 0, errors.New("ReadFloat() is is of buffer")
}

func (buff *TBuffer) ReadString() (string, error) {

	len, err := buff.ReadInt()
	if err != nil {
		return "", err
	}

	if buff.pos+len <= buff.size {
		value := string(buff.data[buff.pos : buff.pos+len])
		buff.pos += len
		return value, nil
	}

	return "", errors.New("ReadString() is is of buffer")
}

func (buff *TBuffer) ReadBlob() ([]byte, error) {

	len, err := buff.ReadInt()
	if err != nil {
		return nil, err
	}

	if buff.pos+len <= buff.size {
		value := buff.data[buff.pos : buff.pos+len]
		buff.pos += len
		return value, nil
	}

	return nil, errors.New("ReadBlob() is is of buffer")
}*/

func (buff *TBuffer) ReadSBlob(data []byte, len int64) error {

	if buff.pos+len <= buff.size {
		copy(data, buff.data[buff.pos:buff.pos+len])
		buff.pos += len
		return nil
	}

	return errors.New("ReadSBlob() is is of buffer")
}

func (buff *TBuffer) writeNumber(value int64) {

	var t [8]byte
	binary.LittleEndian.PutUint64(t[:], uint64(value))

	n := 0
	mask := uint8(0)
	for i := 0; i < 8; i++ {
		if t[i] != 0 {
			mask |= 1 << i
			n++
		}
	}

	buff.Resize(buff.size + int64(1+n))
	buff.data[buff.size] = mask
	buff.size += 1

	for i := 0; i < 8; i++ {
		if t[i] != 0 {
			buff.data[buff.size] = t[i]
			buff.size += 1
		}
	}
}
