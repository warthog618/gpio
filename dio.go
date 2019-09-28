// Copyright © 2017 Kent Gibson <warthog618@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

//
//
// Package gpio provides GPIO access on the Raspberry Pi (rev 2 and later).
//
// Supports simple operations such as:
// - Pin mode/direction (input/output)
// - Pin write (high/low)
// - Pin read (high/low)
// - Pull up/down/off
//
// The package intentionally does not support:
//  - the obsoleted rev 1 PCB (no longer worth the effort)
//  - active low (to prevent confusion this package reflects only the actual hardware levels)
//
// Example of use:
//
// 	gpio.Open()
// 	defer gpio.Close()
//
// 	pin := gpio.NewPin(gpio.J8p7)
// 	pin.Low()
// 	pin.Output()
//
// 	for {
// 		pin.Toggle()
// 		time.Sleep(time.Second)
// 	}
//
// The library uses the raw BCM2835 pin numbers, not the ports as they are mapped
// on the J8 output pins for the Raspberry Pi.
// A mapping from J8 to BCM is provided for those wanting to use the J8 numbering.
//
// See the spec for full details of the BCM2835 controller:
// http://www.raspberrypi.org/wp-content/uploads/2012/02/BCM2835-ARM-Peripherals.pdf
//
package gpio

import (
	"time"
)

// Pin represents a single GPIO pin.
type Pin struct {
	// Immutable fields
	pin         int
	fsel        int
	levelReg    int
	clearReg    int
	setReg      int
	pullReg2711 int
	bank        int
	mask        uint32
	// Mutable fields
	shadow Level
}

// Level represents the high (true) or low (false) level of a Pin.
type Level bool

// Mode defines the IO mode of a Pin.
type Mode int

// Pull defines the pull up/down state of a Pin.
type Pull int

const (
	memLength = 4096

	modeMask uint32 = 7 // pin mode is 3 bits wide
	pullMask uint32 = 3 // pull mode is 2 bits wide
	// BCM2835 pullReg is the same for all pins.
	pullReg2835 = 37
)

// Pin Mode, a pin can be set in Input or Output mode
const (
	Input Mode = iota
	Output
	Alt5
	Alt4
	Alt0
	Alt1
	Alt2
	Alt3
)

// Level of pin, High / Low
const (
	Low  Level = false
	High Level = true
)

// Pull Up / Down / Off
const (
	// Values match bcm pull field.
	PullNone Pull = iota
	PullDown
	PullUp
)

// Convenience mapping from J8 pinouts to BCM pinouts.
const (
	J8p27 = iota
	J8p28
	J8p3
	J8p5
	J8p7
	J8p29
	J8p31
	J8p26
	J8p24
	J8p21
	J8p19
	J8p23
	J8p32
	J8p33
	J8p8
	J8p10
	J8p36
	J8p11
	J8p12
	J8p35
	J8p38
	J8p40
	J8p15
	J8p16
	J8p18
	J8p22
	J8p37
	J8p13
	MaxGPIOPin
)

// GPIO aliases to J8 pins
const (
	GPIO2  = J8p3
	GPIO3  = J8p5
	GPIO4  = J8p7
	GPIO5  = J8p29
	GPIO6  = J8p31
	GPIO7  = J8p26
	GPIO8  = J8p24
	GPIO9  = J8p21
	GPIO10 = J8p19
	GPIO11 = J8p23
	GPIO12 = J8p32
	GPIO13 = J8p33
	GPIO14 = J8p8
	GPIO15 = J8p10
	GPIO16 = J8p36
	GPIO17 = J8p11
	GPIO18 = J8p12
	GPIO19 = J8p35
	GPIO20 = J8p38
	GPIO21 = J8p40
	GPIO22 = J8p15
	GPIO23 = J8p16
	GPIO24 = J8p18
	GPIO25 = J8p22
	GPIO26 = J8p37
	GPIO27 = J8p13
)

// NewPin creates a new pin object.
// The pin number provided is the BCM GPIO number.
func NewPin(pin int) *Pin {
	if len(mem) == 0 {
		panic("GPIO not initialised.")
	}
	if pin < 0 || pin >= MaxGPIOPin {
		return nil
	}

	// Pre-calculate commonly used register addresses and bit masks.

	// Pin fsel register, 0 - 5 depending on pin
	fsel := pin / 10

	// This seems like overkill given the J8 pins are all on the first bank...
	bank := pin / 32
	mask := uint32(1 << uint(pin & 0x1f))

	// Input level register offset (13 / 14 depending on bank)
	levelReg := 13 + bank

	// Clear register, 10 / 11 depending on bank
	clearReg := 10 + bank

	// Set register, 7 / 8 depending on bank
	setReg := 7 + bank

	// Pull register, 57-60 depending on pin
	pullReg := 57 + pin/16

	shadow := Low
	if mem[levelReg]&mask != 0 {
		shadow = High
	}

	return &Pin{
		pin:         pin,
		fsel:        fsel,
		bank:        bank,
		mask:        mask,
		levelReg:    levelReg,
		clearReg:    clearReg,
		pullReg2711: pullReg,
		setReg:      setReg,
		shadow:      shadow,
	}
}

// Input sets pin as Input.
func (pin *Pin) Input() {
	pin.SetMode(Input)
}

// Output sets pin as Output.
func (pin *Pin) Output() {
	pin.SetMode(Output)
}

// High sets pin High.
func (pin *Pin) High() {
	pin.Write(High)
}

// Low sets pin Low.
func (pin *Pin) Low() {
	pin.Write(Low)
}

// Mode returns the mode of the pin in the Function Select register.
func (pin *Pin) Mode() Mode {
	// read Mode and current value
	modeShift := uint(pin.pin % 10) * 3
	return Mode(mem[pin.fsel] >> modeShift & modeMask)
}

// Shadow returns the value of the last write to an output pin or the last read on an input pin.
func (pin *Pin) Shadow() Level {
	return pin.shadow
}

// Pin returns the pin number that this Pin represents.
func (pin *Pin) Pin() int {
	return pin.pin
}

// Toggle pin state
func (pin *Pin) Toggle() {
	if pin.shadow {
		pin.Write(Low)
	} else {
		pin.Write(High)
	}
}

// SetMode sets the pin Mode.
func (pin *Pin) SetMode(mode Mode) {
	// shift for pin mode field within fsel register.
	modeShift := uint(pin.pin % 10) * 3

	memlock.Lock()
	defer memlock.Unlock()

	mem[pin.fsel] = mem[pin.fsel]&^(modeMask<<modeShift) | uint32(mode)<<modeShift
}

// Read pin state (high/low)
func (pin *Pin) Read() (level Level) {
	if (mem[pin.levelReg] & pin.mask) != 0 {
		level = High
	}
	pin.shadow = level
	return
}

// Set pin state (high/low)
func (pin *Pin) Write(level Level) {
	if level == Low {
		mem[pin.clearReg] = pin.mask
	} else {
		mem[pin.setReg] = pin.mask
	}
	pin.shadow = level
}

// SetPull sets the pull up/down mode for a Pin.
// Unlike the mode, the pull value cannot be read back from hardware and
// so must be remembered by the caller.
func (pin *Pin) SetPull(pull Pull) {
	switch chipset {
	case BCM2711:
		pin.setPull2711(pull)
	default:
		pin.setPull2835(pull)
	}
}

func (pin *Pin) setPull2835(pull Pull) {
	clkReg := pin.bank + 38
	memlock.Lock()
	defer memlock.Unlock()

	mem[pullReg2835] = mem[pullReg2835]&^pullMask | uint32(pull)
	// Wait for value to clock in, this is ugly, sorry :(
	// This wait corresponds to at least 150 clock cycles.
	time.Sleep(time.Microsecond)
	mem[clkReg] = pin.mask
	// Wait for value to clock in
	time.Sleep(time.Microsecond)
	mem[pullReg2835] = mem[pullReg2835] &^ pullMask
	mem[clkReg] = 0

}

func (pin *Pin) setPull2711(pull Pull) {
	// 2711 reverses up/down sense
	switch pull {
	case PullUp:
		pull = PullDown
	case PullDown:
		pull = PullUp
	}
	shift := uint(pin.pin & 0x0f) << 1
	memlock.Lock()
	defer memlock.Unlock()
	mem[pin.pullReg2711] = mem[pin.pullReg2711]&^(pullMask<<shift) | uint32(pull)<<shift
}

// PullUp sets the pull state of the pin to PullUp.
func (pin *Pin) PullUp() {
	pin.SetPull(PullUp)
}

// PullDown sets the pull state of the Pin to PullDown.
func (pin *Pin) PullDown() {
	pin.SetPull(PullDown)
}

// PullNone disables pullup/down on pin, leaving it floating.
func (pin *Pin) PullNone() {
	pin.SetPull(PullNone)
}
