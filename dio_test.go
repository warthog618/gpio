// Copyright © 2017 Kent Gibson <warthog618@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

//
//  Test suite for dio module.
//
//	Tests use J8 pins 7 (mostly) and 15 and 16 (for looped tests)
//
package gpio_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/warthog618/gpio"
)

func setupDIO(t *testing.T) {
	assert.Nil(t, gpio.Open())
}

func teardownDIO() {
	gpio.Close()
}

func TestUninitialisedPanic(t *testing.T) {
	assert.Panics(t, func() {
		gpio.NewPin(gpio.J8p7)
	})
}

func TestClosedPanic(t *testing.T) {
	setupDIO(t)
	teardownDIO()
	assert.Panics(t, func() {
		gpio.NewPin(gpio.J8p7)
	})
}

func TestNew(t *testing.T) {
	setupDIO(t)
	defer teardownDIO()
	pin := gpio.NewPin(gpio.MaxGPIOPin)
	assert.Nil(t, pin)
}

func TestRead(t *testing.T) {
	setupDIO(t)
	defer teardownDIO()
	pin := gpio.NewPin(gpio.J8p7)
	// A basic read test - assuming the pin is input and pulled high
	// which is the default state for this pin on a Pi.
	assert.Equal(t, gpio.Input, pin.Mode())
	// Assumes pin is initially pulled up and set as an input.
	assert.Equal(t, gpio.High, pin.Read())
}

func TestMode(t *testing.T) {
	setupDIO(t)
	defer teardownDIO()
	pin := gpio.NewPin(gpio.J8p7)
	assert.Equal(t, gpio.Input, pin.Mode())

	pin.SetMode(gpio.Output)
	assert.Equal(t, gpio.Output, pin.Mode())

	pin.SetMode(gpio.Input)
	assert.Equal(t, gpio.Input, pin.Mode())

	pin.Output()
	assert.Equal(t, gpio.Output, pin.Mode())

	pin.Input()
	assert.Equal(t, gpio.Input, pin.Mode())
}

func TestPull(t *testing.T) {
	setupDIO(t)
	defer teardownDIO()
	pin := gpio.NewPin(gpio.J8p7)
	defer pin.PullUp()
	// A basic read test - using the pull up/down to drive the pin.
	assert.Equal(t, gpio.Input, pin.Mode())
	pin.PullUp()
	pullSettle := time.Microsecond
	time.Sleep(pullSettle)
	assert.Equal(t, gpio.High, pin.Read())
	pin.PullDown()
	time.Sleep(pullSettle)
	assert.Equal(t, gpio.Low, pin.Read())
	pin.SetPull(gpio.PullUp)
	time.Sleep(pullSettle)
	assert.Equal(t, gpio.High, pin.Read())
	// no real way of testing this, but to trick coverage...
	pin.PullNone()
}

func TestPin(t *testing.T) {
	setupDIO(t)
	defer teardownDIO()
	pin := gpio.NewPin(gpio.J8p7)
	assert.Equal(t, gpio.J8p7, pin.Pin())
	pin = gpio.NewPin(gpio.J8p16)
	assert.Equal(t, gpio.J8p16, pin.Pin())
}

func TestWrite(t *testing.T) {
	setupDIO(t)
	defer teardownDIO()
	pin := gpio.NewPin(gpio.J8p7)
	mode := pin.Mode()
	assert.Equal(t, gpio.Input, mode)
	defer pin.SetMode(gpio.Input)

	pin.Write(gpio.Low)
	pin.SetMode(gpio.Output)
	assert.Equal(t, gpio.Output, pin.Mode())
	assert.Equal(t, gpio.Low, pin.Read())

	pin.Write(gpio.High)
	assert.Equal(t, gpio.High, pin.Shadow())
	assert.Equal(t, gpio.High, pin.Read())

	pin.Write(gpio.Low)
	assert.Equal(t, gpio.Low, pin.Shadow())
	assert.Equal(t, gpio.Low, pin.Read())

	pin.High()
	assert.Equal(t, gpio.High, pin.Shadow())
	assert.Equal(t, gpio.High, pin.Read())

	pin.Low()
	assert.Equal(t, gpio.Low, pin.Shadow())
	assert.Equal(t, gpio.Low, pin.Read())
}

// Looped tests require a jumper across Raspberry Pi J8 pins 15 and 16.
func TestWriteLooped(t *testing.T) {
	setupDIO(t)
	defer teardownDIO()
	pinIn := gpio.NewPin(gpio.J8p15)
	pinOut := gpio.NewPin(gpio.J8p16)
	pinIn.SetMode(gpio.Input)
	defer pinOut.SetMode(gpio.Input)
	pinOut.Write(gpio.Low)
	pinOut.SetMode(gpio.Output)
	assert.Equal(t, gpio.Low, pinIn.Read())

	pinOut.Write(gpio.High)
	assert.Equal(t, gpio.High, pinIn.Read())

	pinOut.Write(gpio.Low)
	assert.Equal(t, gpio.Low, pinIn.Read())
}

func TestToggle(t *testing.T) {
	setupDIO(t)
	defer teardownDIO()
	pin := gpio.NewPin(gpio.J8p7)
	defer pin.SetMode(gpio.Input)
	pin.Write(gpio.Low)
	pin.SetMode(gpio.Output)
	assert.Equal(t, gpio.Output, pin.Mode())
	assert.Equal(t, gpio.Low, pin.Read())

	pin.Toggle()
	assert.Equal(t, gpio.High, pin.Shadow())
	assert.Equal(t, gpio.High, pin.Read())

	pin.Toggle()
	assert.Equal(t, gpio.Low, pin.Shadow())
	assert.Equal(t, gpio.Low, pin.Read())
}

// Looped tests require a jumper across Raspberry Pi J8 pins 15 and 16.
func TestToggleLooped(t *testing.T) {
	setupDIO(t)
	defer teardownDIO()
	pinIn := gpio.NewPin(gpio.J8p15)
	pinOut := gpio.NewPin(gpio.J8p16)
	pinIn.SetMode(gpio.Input)
	defer pinOut.SetMode(gpio.Input)
	pinOut.Write(gpio.Low)
	pinOut.SetMode(gpio.Output)
	assert.Equal(t, gpio.Output, pinOut.Mode())
	assert.Equal(t, gpio.Low, pinOut.Read())

	pinOut.Toggle()
	assert.Equal(t, gpio.High, pinIn.Read())

	pinOut.Toggle()
	assert.Equal(t, gpio.Low, pinIn.Read())
}

func BenchmarkRead(b *testing.B) {
	assert.Nil(b, gpio.Open())
	defer gpio.Close()
	pin := gpio.NewPin(gpio.J8p7)
	for i := 0; i < b.N; i++ {
		_ = pin.Read()
	}
}

func BenchmarkWrite(b *testing.B) {
	err := gpio.Open()
	assert.Nil(b, err)
	defer gpio.Close()
	pin := gpio.NewPin(gpio.J8p7)
	for i := 0; i < b.N; i++ {
		pin.Write(gpio.High)
	}
}

func BenchmarkToggle(b *testing.B) {
	assert.Nil(b, gpio.Open())
	defer gpio.Close()
	pin := gpio.NewPin(gpio.J8p7)
	defer pin.SetMode(gpio.Input)
	pin.Write(gpio.Low)
	pin.SetMode(gpio.Output)
	for i := 0; i < b.N; i++ {
		pin.Toggle()
	}
}
