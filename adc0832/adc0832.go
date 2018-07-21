// Copyright © 2018 Kent Gibson <warthog618@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package adc0832

import (
	"sync"
	"time"

	"github.com/warthog618/gpio"
)

// ADC0832 reads ADC values from a connected ADC0832.
// The two data pins, di and do, may be tied and connected to a single GPIO pin.
type ADC0832 struct {
	mu sync.Mutex
	// time between clock edges (i.e. half the cycle time)
	tclk time.Duration
	// time to allow mux to settle after clocking out ODD/SIGN
	tset time.Duration
	clk  *gpio.Pin
	csz  *gpio.Pin
	di   *gpio.Pin
	do   *gpio.Pin
}

// New creates a ADC0832.
func New(tclk, tset time.Duration, clk, csz, di, do uint8) *ADC0832 {
	a := &ADC0832{
		tclk: tclk,
		tset: tset,
		clk:  gpio.NewPin(clk),
		csz:  gpio.NewPin(csz),
		di:   gpio.NewPin(di),
		do:   gpio.NewPin(do),
	}
	// hold ADC reset until needed...
	a.clk.Low()
	a.clk.Output()
	a.csz.High()
	a.csz.Output()
	return a
}

// Close disables the output pins used to drive the ADC.
func (a *ADC0832) Close() {
	a.mu.Lock()
	a.clk.Input()
	a.csz.Input()
	a.di.Input()
	a.mu.Unlock()
}

// Read returns the value read from the ADC.
func (a *ADC0832) Read(ch int) uint8 {
	a.mu.Lock()
	a.csz.High()
	a.clk.Low()
	a.di.High()
	a.di.Output()
	time.Sleep(a.tclk)
	a.csz.Low()

	odd := gpio.Low
	if ch != 0 {
		odd = gpio.High
	}
	a.clockOut(gpio.High) // Start
	a.clockOut(gpio.High) // SGL/DIFZ - signal mode
	a.clockOut(odd)       // ODD/Sign
	// mux settling
	a.di.Input()
	time.Sleep(a.tset)
	a.clk.High()
	// MSB first byte
	var d uint8
	for i := uint(0); i < 8; i++ {
		b := a.clockIn()
		d = d << 1
		if b {
			d = d | 0x01
		}
	}
	// ignore LSB bits - same as MSB just reversed order
	a.csz.High()
	a.mu.Unlock()
	return d
}

// clockIn clocks in a data bit from the ADC on do.
// Assumes clock starts high and ends with the rising edge of the next clock.
func (a *ADC0832) clockIn() gpio.Level {
	time.Sleep(a.tclk)
	a.clk.Low() // ADC writes on the falling edge
	time.Sleep(a.tclk)
	b := a.do.Read()
	a.clk.High()
	return b
}

// clockOut clocks out a data bit to the ADC on di
// Assumes clock starts low and ends with the falling edge of the next clock.
func (a *ADC0832) clockOut(l gpio.Level) {
	a.di.Write(l)
	time.Sleep(a.tclk)
	a.clk.High() // ADC reads on the rising edge
	time.Sleep(a.tclk)
	a.clk.Low()
}
