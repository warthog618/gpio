// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/warthog618/gpio"
)

func init() {
	setCmd.Flags().BoolVarP(&setOpts.ActiveLow, "active-low", "l", false, "treat the line level as active low")
	setCmd.SetHelpTemplate(setCmd.HelpTemplate() + extendedSetHelp)
	rootCmd.AddCommand(setCmd)
}

var (
	setCmd = &cobra.Command{
		Use:     "set <pin1>=<level1>...",
		Short:   "Set the level of a pin or pins",
		Args:    cobra.MinimumNArgs(1),
		RunE:    set,
		Example: "  gppio set J8p15=high J8P7=0",
	}
	setOpts = struct {
		ActiveLow bool
	}{}
)

var extendedSetHelp = `
Pins:
  Pins may be identified by name (J8pXX) or number (0-26).

Levels:
  Levels may be [high|hi|true|1|low|lo|false|0] and are case insensitive.
  
Note that setting a pin forces it into output mode.
`

func set(cmd *cobra.Command, args []string) error {
	ll := []int(nil)
	vv := []gpio.Level(nil)
	for _, arg := range args {
		o, v, err := parseLineLevel(arg)
		if err != nil {
			return err
		}
		ll = append(ll, o)
		vv = append(vv, v)
	}
	err := gpio.Open()
	if err != nil {
		return err
	}
	defer gpio.Close()
	for i, v := range vv {
		pin := gpio.NewPin(ll[i])
		if getOpts.ActiveLow {
			v = !v
		}
		pin.Output()
		pin.Write(v)
	}
	return nil
}

func parseLineLevel(arg string) (int, gpio.Level, error) {
	aa := strings.Split(arg, "=")
	if len(aa) != 2 {
		return 0, gpio.Low, fmt.Errorf("invalid pin<->level mapping: %s", arg)
	}
	o, err := parseOffset(aa[0])
	if err != nil {
		return 0, gpio.Low, err
	}
	v, err := parseLevel(aa[1])
	if err != nil {
		return 0, gpio.Low, err
	}
	return int(o), v, nil
}

func parseLevel(arg string) (gpio.Level, error) {
	if l, ok := levelNames[strings.ToLower(arg)]; ok {
		return l, nil
	}
	return gpio.Low, fmt.Errorf("can't parse level '%s'", arg)
}

var levelNames = map[string]gpio.Level{
	"high":  gpio.High,
	"hi":    gpio.High,
	"true":  gpio.High,
	"1":     gpio.High,
	"low":   gpio.Low,
	"lo":    gpio.Low,
	"false": gpio.Low,
	"0":     gpio.Low,
}
