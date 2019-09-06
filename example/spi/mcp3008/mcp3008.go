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

// This example reads both channels from an MCP3008 connected to the RPI by four
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
	adc := mcp3w0c.NewMCP3008(
		tclk,
		int(cfg.MustGet("clk").Int()),
		int(cfg.MustGet("csz").Int()),
		int(cfg.MustGet("di").Int()),
		int(cfg.MustGet("do").Int()))
	defer adc.Close()
	for ch := 0; ch < 8; ch++ {
		d := adc.Read(ch)
		fmt.Printf("ch%d=0x%04x\n", ch, d)
	}
}

func loadConfig() *config.Config {
	defaultConfig := map[string]interface{}{
		"tclk": "500ns",
		"clk":  gpio.GPIO21,
		"csz":  gpio.GPIO6,
		"di":   gpio.GPIO19,
		"do":   gpio.GPIO26,
	}
	def := dict.New(dict.WithMap(defaultConfig))
	shortFlags := map[byte]string{
		'c': "config-file",
	}
	// highest priority sources first - flags override environment
	cfg := config.New(
		pflag.New(pflag.WithShortFlags(shortFlags)),
		env.New(env.WithEnvPrefix("MCP3008_")),
		config.WithDefault(def))
	cfg.Append(
		blob.NewConfigFile(cfg, "config.file", "mcp3008.json", json.NewDecoder()))
	cfg = cfg.GetConfig("", config.WithMust())
	return cfg
}
