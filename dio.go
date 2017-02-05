/*

Package gpio provides GPIO access on the Raspberry Pi (rev 2 and later).

Supports simple operations such as:
- Pin mode/direction (input/output)
- Pin write (high/low)
- Pin read (high/low)
- Pull up/down/off

The package intentionally does not support:
 - the obsoleted rev 1 PCB (no longer worth the effort)
 - active low (to prevent confusion this package reflects only the actual hardware levels)

Example of use:

	gpio.Open()
	defer gpio.Close()

	pin := gpio.NewPin(gpio.J8_7)
	pin.Low()
	pin.Output()

	for {
		pin.Toggle()
		time.Sleep(time.Second)
	}

The library uses the raw BCM2835 pin numbers, not the ports as they are mapped
on the J8 output pins for the Raspberry Pi.
A mapping from J8 to BCM is provided for those wanting to use the J8 numbering.

See the spec for full details of the BCM2835 controller:
http://www.raspberrypi.org/wp-content/uploads/2012/02/BCM2835-ARM-Peripherals.pdf

*/

package gpio

import (
	"time"
)

type Pin struct {
	// Immutable fields
	pin      uint8
	fsel     uint8
	levelReg uint8
	clearReg uint8
	setReg   uint8
	bank     uint8
	mask     uint32
	// Mutable fields
	shadow Level
}
type Level bool
type Mode uint8
type Pull uint8

const (
	memLength = 4096

	modeMask uint32 = 7 // pin mode is 3 bits wide
	pullMask uint32 = 3 // pull mode is 2 bits wide
	// pullReg is the same for all banks.
	pullReg = 37
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
	High       = true
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
	J8_27 = iota
	J8_28
	J8_3
	J8_5
	J8_7
	J8_29
	J8_31
	J8_26
	J8_24
	J8_21
	J8_19
	J8_23
	J8_32
	J8_33
	J8_8
	J8_10
	J8_36
	J8_11
	J8_12
	J8_35
	J8_38
	J8_40
	J8_15
	J8_16
	J8_18
	J8_22
	J8_37
	J8_13
)

// Create a new pin object.  The pin number provided is the BCM GPIO number.
func NewPin(pin uint8) *Pin {
	if len(mem) == 0 {
		panic("GPIO not initialised.")
	}
	// Pre-calculate commonly used register addresses and bit masks.
	// Pin fsel register, 0 - 5 depending on pin
	fsel := pin / 10
	// This seems like overkill given the J8 pins are all on the first bank...
	bank := pin / 32
	mask := uint32(1 << (pin & 0x1f))
	// Input level register offset (13 / 14 depending on bank)
	levelReg := 13 + bank
	// Clear register, 10 / 11 depending on bank
	clearReg := 10 + bank
	// Set register, 7 / 8 depending on bank
	setReg := 7 + bank

	shadow := Low
	if mem[levelReg]&mask != 0 {
		shadow = High
	}

	return &Pin{pin: pin, fsel: fsel, bank: bank, mask: mask,
		levelReg: levelReg, clearReg: clearReg, setReg: setReg, shadow: shadow}
}

// Set pin as Input
func (pin *Pin) Input() {
	pin.SetMode(Input)
}

// Set pin as Output
func (pin *Pin) Output() {
	pin.SetMode(Output)
}

// Set pin High
func (pin *Pin) High() {
	pin.Write(High)
}

// Set pin Low
func (pin *Pin) Low() {
	pin.Write(Low)
}

// The mode of the pin in the Function Select register.
func (pin *Pin) Mode() Mode {
	// read Mode and current value
	modeShift := (pin.pin % 10) * 3
	return Mode(mem[pin.fsel] >> modeShift & modeMask)
}

// The value of the last write to an output pin or the last read on an input pin.
func (pin *Pin) Shadow() Level {
	return pin.shadow
}

// Toggle pin state
func (pin *Pin) Toggle() {
	if pin.shadow {
		pin.Write(Low)
	} else {
		pin.Write(High)
	}
}

// Set pin Mode
func (pin *Pin) SetMode(mode Mode) {
	// shift for pin mode field within fsel register.
	modeShift := (pin.pin % 10) * 3

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

// Set a given pull up/down mode
// Unlike the mode, the pull value cannot be read back from hardware and
// so must be remembered by the caller.
func (pin *Pin) SetPull(pull Pull) {
	// Pull up/down/off register has offset 38 / 39
	pullClkReg := pin.bank + 38

	memlock.Lock()
	defer memlock.Unlock()

	mem[pullReg] = mem[pullReg]&^pullMask | uint32(pull)
	// Wait for value to clock in, this is ugly, sorry :(
	// This wait corresponds to at least 150 clock cycles.
	time.Sleep(time.Microsecond)
	mem[pullClkReg] = pin.mask
	// Wait for value to clock in
	time.Sleep(time.Microsecond)
	mem[pullReg] = mem[pullReg] &^ pullMask
	mem[pullClkReg] = 0
}

// Pull up pin
func (pin *Pin) PullUp() {
	pin.SetPull(PullUp)
}

// Pull down pin
func (pin *Pin) PullDown() {
	pin.SetPull(PullDown)
}

// Disable pullup/down on pin
func (pin *Pin) PullNone() {
	pin.SetPull(PullNone)
}
