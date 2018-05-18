# gpio

[![GoDoc](https://godoc.org/github.com/warthog618/gpio?status.svg)](https://godoc.org/github.com/warthog618/gpio)
[![Go Report Card](https://goreportcard.com/badge/github.com/warthog618/gpio)](https://goreportcard.com/report/github.com/warthog618/gpio)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/warthog618/gpio/blob/master/LICENSE)

GPIO library for the Raspberry Pi.

gpio is a Go library for accessing [GPIO](http://elinux.org/Rpi_Low-level_peripherals) pins on the [Raspberry Pi](https://en.wikipedia.org/wiki/Raspberry_Pi).

The library was inspired by and borrows from [go-rpio](https://github.com/stianeikeland/go-rpio), which is fast but lacks interrupt support, and [embd](https://github.com/kidoman/embd), which supports interrupts, but uses sysfs for read/write and has a far broader scope than I require.

## Features

Supports the following functionality:

- Pin Mode/Direction (Input / Output)
- Write (High / Low)
- Read (High / Low)
- Pullups (Up / Down / None)
- Watches/Interrupts (Rising/Falling/Both)

## Usage

```go
import "github.com/warthog618/gpio"
```

### Library Initialization

Open memory range for GPIO access in /dev/gpiomem

```go
err := gpio.Open()
```

Cleanup when done

```go
gpio.Close()
```

### Pin Initialization

A Pin object is constructed using the *NewPin* function.
The Pin object is then used for all operations on that pin.
Note that the pin number refers to the BCM GPIO pin, not the physical pin on the Raspberry Pi header.
Pin 4 here is exposed on the pin header as physical pin 7 (J8 7).
Mappings are provided from Raspberry Pi J8 header pin names to BCM GPIO numbers, using the form J8pX.

```go
pin := gpio.NewPin(4)
pin := gpio.NewPin(gpio.J8p7) // Using Raspberry Pi J8 mapping.
```

There is no need to cleanup a pin if you no longer need to use it, unless it has Watches set in which case you should remove the *Watch*.

### Mode

The pin mode controls whether the pin is an input or output.  The existing mode can be read back.

```go
mode := pin.Mode()
pin.Output()               // Set mode to Output
pin.Input()                // Set mode to Input
pin.SetMode(gpio.Output)   // Alternate syntax
```

To prevent output glitches, the pin level can be set using *High*/*Low*/*Write* before the pin is set to Output.

### Input

```go
res := pin.Read()  // Read state from pin (High / Low)
```

### Output

```go
pin.High()              // Set pin High
pin.Low()               // Set pin Low
pin.Toggle()            // Toggle pin (Low -> High -> Low)

pin.Write(gpio.High)    // Alternate syntax
```

Also see example [example/blinker/blinker.go](example/blinker/blinker.go)

### Pullups

Pull up state can be set using:

```go
pin.PullUp()
pin.PullDown()
pin.PullNone()

pin.SetPull(gpio.PullUp)  // Alternate syntax
```

Unlike the Mode, the pull up state cannot be read back from hardware, so there is no *Pull* function.

### Watches

The state of an input pin can be watched and trigger calls to handler functions.

The watch can be on rising or falling edges, or both.

The handler function is passed the triggering pin.

```go
func handler(*Pin) {
  // handle change in pin value
}
pin.Watch(gpio.EdgeFalling,handler)    // Call handler when pin changes from High to Low.

pin.Watch(gpio.EdgeRising,handler)     // Call handler when pin changes from Low to High.

pin.Watch(gpio.EdgeBoth,handler)       // Call handler when pin changes
```

A watch can be removed using the *Unwatch* function.

```go
pin.Unwatch()
```

## Examples

Refer to the [examples](example) for more examples of usage.

Examples can be cross-compiled from other platforms using

```sh
GOOS=linux GOARCH=arm GOARM=6 go build
```

## Tests

The library is fully tested, other than some error cases that are difficult to test.

The tests are intended to be run on a Raspberry Pi with J8 pin 7 floating and with pins 15 and 16 tied together, possibly using a jumper across the header.  The tests set J8 pin 16 to an output so **DO NOT** run them on hardware where that pin is being externally driven.

Tests have been run successfully on Raspberry Pi B (Rev 1 and Rev 2), B+, Pi2 B, and Pi Zero W.  The library should also work on other Raspberry Pi variants, I just don't have any available to test.

The tests can be cross-compiled from other platforms using

```sh
GOOS=linux GOARCH=arm GOARM=6 go test -c
```

Later Pis can also use ARM7 (GOARM=7).

### Benchmarks

The tests include benchmarks on reads and writes.  Reading pin levels through sysfs is provided for comparison.

These are the results from a Raspberry Pi B(Rev1)

```sh
$ ./gpio.test -test.bench=.*
PASS
BenchmarkRead                 5000000         240 ns/op
BenchmarkWrite               20000000        81.9 ns/op
BenchmarkToggle              20000000        97.1 ns/op
BenchmarkSysfsRead             100000       11549 ns/op
BenchmarkSysfsWrite             50000       26592 ns/op
BenchmarkSysfsToggle            50000       25414 ns/op
BenchmarkInterruptLatency        1000     1092871 ns/op
```

## Prerequisites

The library assumes Linux, and has been tested on Raspbian Jessie and Stretch.

The library targets all models of the Raspberry Pi.  Note that the Raspberry Pi Model B Rev 1.0 has different pinouts, so the J8 mappings are incorrect for that particular revision.

This library utilizes /dev/gpiomem, which must be available to the current user.  This is generally available in recent Raspian releases.

The library also utilizes the sysfs GPIO to support interrupts on changes to input pin values.  The sysfs is not used to access the pin values, as the gpiomem approach is orders of magnitude faster (refer to the benchmarks).
