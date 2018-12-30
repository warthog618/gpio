// Copyright © 2018 Kent Gibson <warthog618@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp3008

import (
	"sync"
	"time"

	"github.com/warthog618/gpio"
)

// MCP3008 reads ADC values from a connected MCP3008.
// The two data pins, di and do, may be tied and connected to a single GPIO pin.
type MCP3008 struct {
	mu sync.Mutex
	// time between clock edges (i.e. half the cycle time)
	tclk time.Duration
	clk  *gpio.Pin
	csz  *gpio.Pin
	di   *gpio.Pin
	do   *gpio.Pin
}

// New creates a MCP3008.
func New(tclk time.Duration, clk, csz, di, do uint8) *MCP3008 {
	adc := &MCP3008{
		tclk: tclk,
		clk:  gpio.NewPin(clk),
		csz:  gpio.NewPin(csz),
		di:   gpio.NewPin(di),
		do:   gpio.NewPin(do),
	}
	// hold ADC reset until needed...
	adc.clk.Low()
	adc.clk.Output()
	adc.csz.High()
	adc.csz.Output()
	return adc
}

// Close disables the output pins used to drive the ADC.
func (adc *MCP3008) Close() {
	adc.mu.Lock()
	adc.clk.Input()
	adc.csz.Input()
	adc.di.Input()
	adc.mu.Unlock()
}

// Read returns the value read from the ADC.
func (adc *MCP3008) Read(ch int) uint16 {
	adc.mu.Lock()
	adc.csz.High()
	adc.clk.Low()
	adc.di.High()
	adc.di.Output()
	time.Sleep(adc.tclk)
	adc.csz.Low()

	adc.clockOut(gpio.High) // Start
	adc.clockOut(gpio.High) // SGL/DIFFZ - signal mode
	for i := 2; i >= 0; i-- {
		d := gpio.Low
		if (ch >> uint(i) & 0x01) == 0x01 {
			d = gpio.High
		}
		adc.clockOut(d)
	}
	// mux settling
	adc.di.Input()
	time.Sleep(adc.tclk)
	adc.clk.High()
	adc.clockIn() // null bit
	var d uint16
	for i := uint(0); i < 10; i++ {
		d = d << 1
		if adc.clockIn() {
			d = d | 0x01
		}
	}
	adc.csz.High()
	adc.mu.Unlock()
	return d
}

// clockIn clocks in a data bit from the ADC on do.
// Assumes clock starts high and ends with the rising edge of the next clock.
func (adc *MCP3008) clockIn() gpio.Level {
	time.Sleep(adc.tclk)
	adc.clk.Low() // ADC writes on the falling edge
	time.Sleep(adc.tclk)
	b := adc.do.Read()
	adc.clk.High()
	return b
}

// clockOut clocks out a data bit to the ADC on di
// Assumes clock starts low and ends with the falling edge of the next clock.
func (adc *MCP3008) clockOut(l gpio.Level) {
	adc.di.Write(l)
	time.Sleep(adc.tclk)
	adc.clk.High() // ADC reads on the rising edge
	time.Sleep(adc.tclk)
	adc.clk.Low()
}
