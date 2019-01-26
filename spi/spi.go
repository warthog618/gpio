// Copyright © 2018 Kent Gibson <warthog618@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package spi

import (
	"sync"
	"time"

	"github.com/warthog618/gpio"
)

// SPI resprents a device connected to the Raspberry Pi via an SPI bus using 3 or 4 GPIO lines.
// Depending on the device, the two data pins, Mosi and Miso, may be tied and connected to a single GPIO pin.
// This is the basis for bit bashed SPI interfaces using GPIO pins.
// It is not related to the SPI device drivers provided by Linux.
type SPI struct {
	Mu sync.Mutex
	// time between clock edges (i.e. half the cycle time)
	Tclk time.Duration
	Sclk *gpio.Pin
	Ssz  *gpio.Pin
	Mosi *gpio.Pin
	Miso *gpio.Pin
}

// New creates a SPI.
func New(tclk time.Duration, sclk, ssz, mosi, miso uint8) *SPI {
	spi := &SPI{
		Tclk: tclk,
		Sclk: gpio.NewPin(sclk),
		Ssz:  gpio.NewPin(ssz),
		Mosi: gpio.NewPin(mosi),
		Miso: gpio.NewPin(miso),
	}
	// hold SPI reset until needed...
	spi.Sclk.Low()
	spi.Sclk.Output()
	spi.Ssz.High()
	spi.Ssz.Output()
	return spi
}

// Close disables the output pins used to drive the SPI device.
func (spi *SPI) Close() {
	spi.Mu.Lock()
	spi.Sclk.Input()
	spi.Ssz.Input()
	spi.Mosi.Input()
	spi.Mu.Unlock()
}

// ClockIn clocks in a data bit from the SPI device on Miso.
// Assumes clock starts high and ends with the rising edge of the next clock.
// Assumes caller already holds the Mu lock.
func (spi *SPI) ClockIn() gpio.Level {
	time.Sleep(spi.Tclk)
	spi.Sclk.Low() // SPI device writes on the falling edge
	time.Sleep(spi.Tclk)
	b := spi.Miso.Read()
	spi.Sclk.High()
	return b
}

// ClockOut clocks out a data bit to the SPI device on Mosi.
// Assumes clock starts low and ends with the falling edge of the next clock.
// Assumes caller already holds the Mu lock.
func (spi *SPI) ClockOut(l gpio.Level) {
	spi.Mosi.Write(l)
	time.Sleep(spi.Tclk)
	spi.Sclk.High() // SPI device reads on the rising edge
	time.Sleep(spi.Tclk)
	spi.Sclk.Low()
}
