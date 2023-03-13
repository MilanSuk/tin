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
	"os"
	"sync/atomic"
	"time"
)

func OsTicks() int64 {
	return time.Now().UnixMilli()
}

func OsIsTicksIn(start_ticks int64, delay_ms int64) bool {
	return (start_ticks + delay_ms) > OsTicks()
}

func OsTime() float64 {
	return float64(OsTicks()) / 1000
}

func OsTimeZone() int {
	_, zn := time.Now().Zone()
	return zn / (24 * 60 * 60)
}

func OsFileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func OsFileBytes(filename string) int64 {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return -1
	}
	return info.Size()
}

func OsFileRemove(filename string) error {
	err := os.Remove(filename)
	if os.IsNotExist(err) {
		return err
	}
	return err
}

// Ternary operator
func OsTrn(question bool, ret_true int, ret_false int) int {
	if question {
		return ret_true
	}
	return ret_false
}

func OsTrnFloat(question bool, ret_true float64, ret_false float64) float64 {
	if question {
		return ret_true
	}
	return ret_false
}
func OsTrnString(question bool, ret_true string, ret_false string) string {
	if question {
		return ret_true
	}
	return ret_false
}

func OsTrnBool(question bool, ret_true bool, ret_false bool) bool {
	if question {
		return ret_true
	}
	return ret_false
}

func OsMax(x, y int) int {
	if x < y {
		return y
	}
	return x
}
func OsMin(x, y int) int {
	if x > y {
		return y
	}
	return x
}
func OsClamp(v, min, max int) int {
	return OsMin(OsMax(v, min), max)
}
func OsClamp32(v, min, max int32) int32 {
	return OsMin32(OsMax32(v, min), max)
}

func OsAbs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func OsMax32(x, y int32) int32 {
	if x < y {
		return y
	}
	return x
}
func OsMin32(x, y int32) int32 {
	if x > y {
		return y
	}
	return x
}

func OsAbs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func OsMaxFloat(x, y float64) float64 {
	if x < y {
		return y
	}
	return x
}
func OsMinFloat(x, y float64) float64 {
	if x > y {
		return y
	}
	return x
}

func OsClampFloat(v, min, max float64) float64 {
	return OsMinFloat(OsMaxFloat(v, min), max)
}

func OsFAbs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func OsRoundDown(v float64) float64 {
	return float64(int64(v))
}
func OsRoundUp(v float64) int64 {
	if v > OsRoundDown(v) {
		return int64(v + 1)
	}
	return int64(v)
}

func OsRoundHalf(v float64) float64 {
	return OsRoundDown(v + OsTrnFloat(v < 0, -0.5, 0.5))
}

type OsThread struct {
	sync atomic.Uint32 //0=run, 1=end, 2=finished
}

func (thread *OsThread) Is() bool {
	return thread.sync.Load() == 0
}

func (thread *OsThread) End() {
	thread.sync.Store(2)
}

func (thread *OsThread) Wait() {
	thread.sync.Store(1)

	//wait
	for thread.sync.Load() != 2 {
		time.Sleep(1 * time.Millisecond)
	}
}
