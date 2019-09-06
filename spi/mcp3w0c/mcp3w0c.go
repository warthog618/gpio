// Copyright © 2018 Kent Gibson <warthog618@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// Package mcp3w0c provides device drivers for MCP3004/3008/3204/3208 SPI ADCs.
package mcp3w0c

import (
	"time"

	"github.com/warthog618/gpio"
	"github.com/warthog618/gpio/spi"
)

// MCP3w0c reads ADC values from a connected Microchip MCP3xxx family device.
// Supported variants are MCP3004/3008/3204/3208.
// The w indicates the width of the device (0 => 10, 2 => 12)
// and the c the number of channels.
// The two data pins, Mosi and Miso, may be tied and connected to a single GPIO pin.
type MCP3w0c struct {
	spi.SPI
	width uint
}

// New creates a MCP3w0c.
func New(tclk time.Duration, sclk, ssz, mosi, miso int, width uint) *MCP3w0c {
	return &MCP3w0c{*spi.New(tclk, sclk, ssz, mosi, miso), width}
}

// NewMCP3008 creates a MCP3008.
func NewMCP3008(tclk time.Duration, sclk, ssz, mosi, miso int) *MCP3w0c {
	return &MCP3w0c{*spi.New(tclk, sclk, ssz, mosi, miso), 10}
}

// NewMCP3208 creates a MCP3208.
func NewMCP3208(tclk time.Duration, sclk, ssz, mosi, miso int) *MCP3w0c {
	return &MCP3w0c{*spi.New(tclk, sclk, ssz, mosi, miso), 12}
}

// Read returns the value of a single channel read from the ADC.
func (adc *MCP3w0c) Read(ch int) uint16 {
	return adc.read(ch, gpio.High)
}

// ReadDifferential returns the value of a differential pair read from the ADC.
func (adc *MCP3w0c) ReadDifferential(ch int) uint16 {
	return adc.read(ch, gpio.Low)
}

func (adc *MCP3w0c) read(ch int, sgl gpio.Level) uint16 {
	adc.Mu.Lock()
	adc.Ssz.High()
	adc.Sclk.Low()
	adc.Mosi.High()
	adc.Mosi.Output()
	time.Sleep(adc.Tclk)
	adc.Ssz.Low()

	adc.ClockOut(gpio.High) // Start
	adc.ClockOut(sgl)       // SGL/DIFFZ
	for i := 2; i >= 0; i-- {
		d := gpio.Low
		if (ch >> uint(i) & 0x01) == 0x01 {
			d = gpio.High
		}
		adc.ClockOut(d)
	}
	// mux settling
	adc.Mosi.Input()
	time.Sleep(adc.Tclk)
	adc.Sclk.High()
	adc.ClockIn() // null bit
	var d uint16
	for i := uint(0); i < adc.width; i++ {
		d = d << 1
		if adc.ClockIn() {
			d = d | 0x01
		}
	}
	adc.Ssz.High()
	adc.Mu.Unlock()
	return d
}
