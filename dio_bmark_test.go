// Copyright © 2017 Kent Gibson <warthog618@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

//
//  Test suite for dio module.
//
//	Tests use J8 pins 7 (mostly) and 15 and 16 (for looped tests)
//
package gpio

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkSysfsRead(b *testing.B) {
	assert.Nil(b, Open())
	defer Close()
	pin := NewPin(J8p7)
	// setup sysfs
	err := export(pin)
	assert.Nil(b, err)
	defer unexport(pin)
	f, err := openValue(pin)
	assert.Nil(b, err)
	defer f.Close()
	r := make([]byte, 1)
	r[0] = 0
	for i := 0; i < b.N; i++ {
		f.Read(r)
	}
}

func BenchmarkSysfsWrite(b *testing.B) {
	assert.Nil(b, Open())
	defer Close()
	pin := NewPin(J8p7)
	// setup sysfs
	err := export(pin)
	assert.Nil(b, err)
	defer unexport(pin)
	f, err := openValue(pin)
	assert.Nil(b, err)
	defer f.Close()
	r := "0"
	for i := 0; i < b.N; i++ {
		f.WriteString(r)
	}
}

func BenchmarkSysfsToggle(b *testing.B) {
	assert.Nil(b, Open())
	defer Close()
	pin := NewPin(J8p7)
	// setup sysfs
	err := export(pin)
	assert.Nil(b, err)
	defer unexport(pin)
	f, err := openValue(pin)
	assert.Nil(b, err)
	defer f.Close()
	r := "0"
	for i := 0; i < b.N; i++ {
		if r == "0" {
			r = "1"
		} else {
			r = "0"
		}
		f.WriteString(r)
	}
}
