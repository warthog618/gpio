// Copyright © 2017 Kent Gibson <warthog618@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build linux

package gpio

import (
	"errors"
	"os"
	"reflect"
	"sync"
	"syscall"
	"unsafe"
)

// Arrays for 8 / 32 bit access to memory and a semaphore for write locking
var (
	// The memlock covers read/modify/write access to the mem block.
	// Individual reads and writes can skip the lock on the assumption that
	// concurrent register writes are atomic. e.g. Read, Write and Mode.
	memlock sync.Mutex
	mem     []uint32
	mem8    []uint8
)

// Open and memory map GPIO memory range from /dev/gpiomem .
// Some reflection magic is used to convert it to a unsafe []uint32 pointer
func Open() (err error) {
	if len(mem) != 0 {
		return ErrAlreadyOpen
	}
	file, err := os.OpenFile(
		"/dev/gpiomem",
		os.O_RDWR|os.O_SYNC,
		0)

	if err != nil {
		return
	}
	defer file.Close()

	memlock.Lock()
	defer memlock.Unlock()

	// Memory map GPIO registers to byte array
	mem8, err = syscall.Mmap(
		int(file.Fd()),
		0,
		memLength,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED)

	if err != nil {
		return
	}

	// Convert mapped byte memory to unsafe []uint32 pointer, adjust length as needed
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&mem8))
	header.Len /= 4 // (32 bit = 4 bytes)
	header.Cap /= 4

	mem = *(*[]uint32)(unsafe.Pointer(&header))

	return nil
}

// Close removes the interrupt handlers and unmaps GPIO memory
func Close() error {
	memlock.Lock()
	defer memlock.Unlock()
	closeInterrupts()
	mem = make([]uint32, 0)
	return syscall.Munmap(mem8)
}

var (
	// ErrAlreadyOpen indicates the mem is already open.
	ErrAlreadyOpen = errors.New("already open")
)
