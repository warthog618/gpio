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
	"github.com/warthog618/gpio/spi/mcp3w0c"
)

// This example reads both channels from an MCP3208 connected to the RPI by four
// data lines - SSZ, SCLK, MOSI, and MISO. The default pin assignments are defined in
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
	adc := mcp3w0c.NewMCP3208(
		tclk,
		cfg.MustGet("sclk").Int(),
		cfg.MustGet("ssz").Int(),
		cfg.MustGet("mosi").Int(),
		cfg.MustGet("miso").Int())
	defer adc.Close()
	for ch := 0; ch < 8; ch++ {
		d := adc.Read(ch)
		fmt.Printf("ch%d=0x%04x (%08b)\n", ch, d, d>>4)
	}
}

func loadConfig() *config.Config {
	defaultConfig := map[string]interface{}{
		"tclk": "500ns",
		"sclk": gpio.GPIO24,
		"ssz":  gpio.GPIO17,
		"mosi": gpio.GPIO27,
		"miso": gpio.GPIO22,
	}
	def := dict.New(dict.WithMap(defaultConfig))
	cfg := config.New(
		pflag.New(pflag.WithFlags(
			[]pflag.Flag{{Short: 'c', Name: "config-file"}})),
		env.New(env.WithEnvPrefix("MCP3208_")),
		config.WithDefault(def))
	cfg.Append(
		blob.NewConfigFile(cfg, "config.file", "mcp3208.json", json.NewDecoder()))
	cfg = cfg.GetConfig("", config.WithMust())
	return cfg
}
