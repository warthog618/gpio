// Copyright © 2017 Kent Gibson <warthog618@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// Test suite for mem module.
package gpio_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/warthog618/gpio"
)

func TestOpen(t *testing.T) {
	assert.Nil(t, gpio.Open())
	defer gpio.Close()
}

func TestOpenOpened(t *testing.T) {
	assert.Nil(t, gpio.Open())
	defer gpio.Close()
	assert.NotNil(t, gpio.Open())
}

func TestReOpen(t *testing.T) {
	assert.Nil(t, gpio.Open())
	gpio.Close()
	assert.Nil(t, gpio.Open())
	defer gpio.Close()
}
