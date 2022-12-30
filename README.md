# gpio

[![Build Status](https://img.shields.io/github/actions/workflow/status/warthog618/gpio/go.yml?logo=github&branch=master)](https://github.com/warthog618/gpio/actions/workflows/go.yml)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/warthog618/gpio)
[![Go Report Card](https://goreportcard.com/badge/github.com/warthog618/gpio)](https://goreportcard.com/report/github.com/warthog618/gpio)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/warthog618/gpio/blob/master/LICENSE)

GPIO library for the Raspberry Pi.

**gpio** is a Go library for accessing
[GPIO](http://elinux.org/Rpi_Low-level_peripherals) pins on the [Raspberry
Pi](https://en.wikipedia.org/wiki/Raspberry_Pi).

The library was inspired by and borrows from
[go-rpio](https://github.com/stianeikeland/go-rpio), which is fast but lacks
interrupt support, and [embd](https://github.com/kidoman/embd), which supports
interrupts, but uses sysfs for read/write and has a far broader scope than I
require.

## :warning: Deprecation Warning :warning:

This library relies on the sysfs GPIO interface which is deprecated in the Linux
kernel and is due for removal during 2020.  Without sysfs, the watch/interrupt
features of this library will no longer work.

The sysfs GPIO interface has been superceded in the Linux kernel by the GPIO
character device.  The newer API is sufficiently different that reworking this
library to use that API is not practical.  Instead I have written a new library,
[**gpiod**](https://github.com/warthog618/gpiod), that provides the same
functionality as this library but using the GPIO character device.

There are a couple of downsides to switching to **gpiod**:

- The API is quite different, mostly due to the differences in the underlying
  APIs, so it is not a plugin replacement - you will need to do some code
  rework.
- It is also slightly slower for both read and write as all hardware access is
  now via kernel calls rather than via hardware registers.  However, if that is
  an issue for you then you probably should be writing a kernel device driver
  for your use case rather than trying to do something in userspace.
- It requires a recent Linux kernel for full functionality.  While the GPIO
  character device has been around since v4.8, the bias and reconfiguration
  capabilities required to provide functional equivalence to **gpio** were only
  added in v5.5.

There are several benefits in the switch:

- **gpiod** is not Raspberry Pi specific, but will work on any platform where
  the GPIO chip is supported by the Linux kernel, so code using **gpiod** is
  more portable.
- **gpio** writes directly to hardware registers so it can conflict with other
  kernel drivers.  The **gpiod** accesses the hardware internally using the same
  interfaces as other kernel drivers and so should play nice with them.
- **gpiod** supports Linux GPIO line labels, so you can find your line by name
  (assuming it has been named by device-tree).
- and of course, it will continue to work beyond 2020.

I've already ported all my projects that were using **gpio** to **gpiod** and
strongly suggest that you do the same.

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

A Pin object is constructed using the *NewPin* function. The Pin object is then
used for all operations on that pin. Note that the pin number refers to the BCM
GPIO pin, not the physical pin on the Raspberry Pi header. Pin 4 here is exposed
on the pin header as physical pin 7 (J8 7). Mappings are provided from Raspberry
Pi J8 header pin names to BCM GPIO numbers, using the form J8pX.

```go
pin := gpio.NewPin(4)
pin := gpio.NewPin(gpio.J8p7) // Using Raspberry Pi J8 mapping.
```

There is no need to cleanup a pin if you no longer need to use it, unless it has
Watches set in which case you should remove the *Watch*.

### Mode

The pin mode controls whether the pin is an input or output.  The existing mode
can be read back.

```go
mode := pin.Mode()
pin.Output()               // Set mode to Output
pin.Input()                // Set mode to Input
pin.SetMode(gpio.Output)   // Alternate syntax
```

To prevent output glitches, the pin level can be set using *High*/*Low*/*Write*
before the pin is set to Output.

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

## Tools

A command line utility, **gppiio**, is provided to allow manual and scripted
control of GPIO pins:

```sh
$ ./gppiio
gppiio is a utility to control Raspberry Pi GPIO pins

Usage:
  gppiio [flags]
  gppiio [command]

Available Commands:
  detect      Identify the GPIO chip
  get         Read the level of a pin or pins
  help        Help about any command
  mode        Read the functional mode of a pin or pins
  mon         Monitor the level of a pin or pins
  pull        Set the pull direction of a pin or pins
  set         Set the level of a pin or pins
  version     Display the version

Flags:
  -h, --help      help for gppiio

Use "gppiio [command] --help" for more information about a command.
```

## Examples

Refer to the [examples](example) for more examples of usage.

Examples can be cross-compiled from other platforms using

```sh
GOOS=linux GOARCH=arm GOARM=6 go build
```

## Tests

The library is fully tested, other than some error cases that are difficult to test.

The tests are intended to be run on a Raspberry Pi with J8 pin 7 floating and
with pins 15 and 16 tied together, possibly using a jumper across the header.
The tests set J8 pin 16 to an output so **DO NOT** run them on hardware where
that pin is being externally driven.

Tests have been run successfully on Raspberry Pi B (Rev 1 and Rev 2), B+, Pi2 B,
Pi4 B, and Pi Zero W.  The library should also work on other Raspberry Pi
variants, I just don't have any available to test.

The tests can be cross-compiled from other platforms using

```sh
GOOS=linux GOARCH=arm GOARM=6 go test -c
```

Later Pis can also use ARM7 (GOARM=7).

### Benchmarks

The tests include benchmarks on reads and writes.  Reading pin levels through sysfs is provided for comparison.

These are the results from a Raspberry Pi Zero W built with Go 1.13:

```sh
$ ./gpio.test -test.bench=.*
goos: linux
goarch: arm
pkg: github.com/warthog618/gpio
BenchmarkRead                  9485052           124 ns/op
BenchmarkWrite                18478959          58.8 ns/op
BenchmarkToggle               16695492          72.4 ns/op
BenchmarkInterruptLatency         2348        453248 ns/op
BenchmarkSysfsRead               32983         31004 ns/op
BenchmarkSysfsWrite              17192         69840 ns/op
BenchmarkSysfsToggle             17341         62962 ns/op

PASS
```

## Prerequisites

The library assumes Linux, and has been tested on Raspbian Jessie, Stretch and Buster.

The library targets all models of the Raspberry Pi, upt to and including the Pi
4B.  Note that the Raspberry Pi Model B Rev 1.0 has different pinouts, so the J8
mappings are incorrect for that particular revision.

This library utilizes /dev/gpiomem, which must be available to the current user.
This is generally available in recent Raspian releases.

The library also utilizes the sysfs GPIO to support interrupts on changes to
input pin values.  The sysfs is not used to access the pin values, as the
gpiomem approach is orders of magnitude faster (refer to the benchmarks).
