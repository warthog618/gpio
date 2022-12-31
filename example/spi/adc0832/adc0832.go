// Copyright © 2018 Kent Gibson <warthog618@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/warthog618/config"
	"github.com/warthog618/config/blob"
	"github.com/warthog618/config/blob/decoder/json"
	"github.com/warthog618/config/dict"
	"github.com/warthog618/config/env"
	"github.com/warthog618/config/pflag"
	"github.com/warthog618/gpio"
	"github.com/warthog618/gpio/spi/adc0832"
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
	tclk := cfg.MustGet("tclk").Duration()
	tset := cfg.MustGet("tset").Duration()
	if tset < tclk {
		tset = tclk
	}
	a := adc0832.New(
		tclk,
		tset,
		cfg.MustGet("clk").Int(),
		cfg.MustGet("csz").Int(),
		cfg.MustGet("di").Int(),
		cfg.MustGet("do").Int())
	defer a.Close()
	ch0 := a.Read(0)
	ch1 := a.Read(1)
	fmt.Printf("ch0=0x%02x, ch1=0x%02x\n", ch0, ch1)
}

func loadConfig() *config.Config {
	defaultConfig := map[string]interface{}{
		"tclk": "2500ns",
		"tset": "2500ns", // should be at least tclk - enforced in main
		"clk":  gpio.GPIO6,
		"csz":  gpio.GPIO5,
		"di":   gpio.GPIO19,
		"do":   gpio.GPIO13,
	}
	def := dict.New(dict.WithMap(defaultConfig))
	cfg := config.New(
		pflag.New(pflag.WithFlags(
			[]pflag.Flag{{Short: 'c', Name: "config-file"}})),
		env.New(env.WithEnvPrefix("ADC0832_")),
		config.WithDefault(def))
	cfg.Append(
		blob.NewConfigFile(cfg, "config.file", "adc0832.json", json.NewDecoder()))
	cfg = cfg.GetConfig("", config.WithMust)
	return cfg
}
