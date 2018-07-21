// Copyright © 2018 Kent Gibson <warthog618@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/warthog618/config"
	"github.com/warthog618/config/dict"
	"github.com/warthog618/config/env"
	"github.com/warthog618/config/json"
	"github.com/warthog618/config/pflag"
	"github.com/warthog618/gpio"
)

// This example reads both channels from an ADC0832 connected to the RPI by four
// data lines - CSZ, CLK, DI, and DO. The default pin assignments are defined in
// loadConfig, but can be altered via configuration (env, flag or config file).
// The DI and DO may be tied to reduce the pin count by one, though I prefer to
// keep the two separate to remove the chance of accidental conflict.
// All pins other than DO are outputs so do not run this example on a board
// where those pins serve other purposes.
func main() {
	cfg := loadConfig()
	err := gpio.Open()
	if err != nil {
		panic(err)
	}
	defer gpio.Close()
	tclk := cfg.GetDuration("tclk")
	tset := cfg.GetDuration("tset")
	if tset < tclk {
		tset = tclk
	}
	a := New(
		tclk,
		tset,
		uint8(cfg.GetUint("clk")),
		uint8(cfg.GetUint("csz")),
		uint8(cfg.GetUint("di")),
		uint8(cfg.GetUint("do")))
	defer a.Close()
	ch0 := a.Read(0)
	ch1 := a.Read(1)
	fmt.Printf("ch0=0x%02x, ch1=0x%02x\n", ch0, ch1)
}

// Config defines the minimal configuration interface
type Config interface {
	GetDuration(k string) time.Duration
	GetUint(k string) uint64
}

func loadConfig() Config {
	defaultConfig := map[string]interface{}{
		"tclk": "2500ns",
		"tset": "2500ns", // should be at least tclk - enforced in main
		"clk":  gpio.GPIO6,
		"csz":  gpio.GPIO5,
		"di":   gpio.GPIO19,
		"do":   gpio.GPIO13,
	}
	def := dict.New(dict.WithMap(defaultConfig))
	shortFlags := map[byte]string{
		'c': "config-file",
	}
	fget, err := pflag.New(pflag.WithShortFlags(shortFlags))
	if err != nil {
		panic(err)
	}
	// environment next
	eget, err := env.New(env.WithEnvPrefix("ADC0832_"))
	if err != nil {
		panic(err)
	}
	// highest priority sources first - flags override environment
	sources := config.NewStack(fget, eget)
	cfg := config.NewConfig(config.Decorate(sources, config.WithDefault(def)))

	// config file may be specified via flag or env, so check for it
	// and if present add it with lower priority than flag and env.
	configFile, err := cfg.GetString("config.file")
	if err == nil {
		// explicitly specified config file - must be there
		jget, err := json.New(json.FromFile(configFile))
		if err != nil {
			panic(err)
		}
		sources.Append(jget)
	} else {
		// implicit and optional default config file
		jget, err := json.New(json.FromFile("adc0832.json"))
		if err == nil {
			sources.Append(jget)
		} else {
			if _, ok := err.(*os.PathError); !ok {
				panic(err)
			}
		}
	}
	m := cfg.GetMust("", config.WithPanic())
	return m
}

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
