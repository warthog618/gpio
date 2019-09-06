// Copyright © 2018 Kent Gibson <warthog618@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package adc0832

import (
	"time"

	"github.com/warthog618/gpio"
	"github.com/warthog618/gpio/spi"
)

// ADC0832 reads ADC values from a connected ADC0832.
// The two data pins, di and do, may be tied and connected to a single GPIO pin.
type ADC0832 struct {
	spi.SPI
	// time to allow mux to settle after clocking out ODD/SIGN
	tset time.Duration
}

// New creates a ADC0832.
func New(tclk, tset time.Duration, sclk, ssz, mosi, miso int) *ADC0832 {
	return &ADC0832{*spi.New(tclk, sclk, ssz, mosi, miso), tset}
}

// Read returns the value of a single channel read from the ADC.
func (adc *ADC0832) Read(ch int) uint8 {
	return adc.read(ch, gpio.High)
}

// ReadDifferential returns the value of a differential pair read from the ADC.
func (adc *ADC0832) ReadDifferential(ch int) uint8 {
	return adc.read(ch, gpio.Low)
}

func (adc *ADC0832) read(ch int, sgl gpio.Level) uint8 {
	adc.Mu.Lock()
	adc.Ssz.High()
	adc.Sclk.Low()
	adc.Mosi.High()
	adc.Mosi.Output()
	time.Sleep(adc.Tclk)
	adc.Ssz.Low()

	odd := gpio.Low
	if ch != 0 {
		odd = gpio.High
	}
	adc.ClockOut(gpio.High) // Start
	adc.ClockOut(sgl)       // SGL/DIFZ
	adc.ClockOut(odd)       // ODD/Sign
	// mux settling
	adc.Mosi.Input()
	time.Sleep(adc.tset)
	adc.Sclk.High()
	// MSB first byte
	var d uint8
	for i := uint(0); i < 8; i++ {
		b := adc.ClockIn()
		d = d << 1
		if b {
			d = d | 0x01
		}
	}
	// ignore LSB bits - same as MSB just reversed order
	adc.Ssz.High()
	adc.Mu.Unlock()
	return d
}
