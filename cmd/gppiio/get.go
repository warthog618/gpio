// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/warthog618/gpio"
)

func init() {
	getCmd.Flags().BoolVarP(&getOpts.All, "all", "a", false, "get the levels of all lines")
	getCmd.Flags().BoolVarP(&getOpts.ActiveLow, "active-low", "l", false, "treat the line level as active low")
	getCmd.Flags().BoolVarP(&getOpts.Short, "short", "s", false, "single line output format")
	getCmd.SetHelpTemplate(getCmd.HelpTemplate() + extendedGetHelp)
	rootCmd.AddCommand(getCmd)
}

var (
	getCmd = &cobra.Command{
		Use:     "get <pin1>...",
		Short:   "Read the level of a pin or pins",
		Example: "  gppio get 23 J8p15",
		PreRunE: preget,
		RunE:    get,
	}
	getOpts = struct {
		ActiveLow bool
		Short     bool
		All       bool
	}{}
)

var extendedGetHelp = `
Pins:
  Pins may be identified by name (J8pXX) or number (0-26).

Note that reading a pin forces it into input mode.
`

func preget(cmd *cobra.Command, args []string) error {
	if !getOpts.All {
		return cobra.MinimumNArgs(1)(cmd, args)
	}
	return nil
}

func get(cmd *cobra.Command, args []string) (err error) {
	var oo []int
	if getOpts.All {
		if len(oo) == 0 {
			oo = make([]int, gpio.MaxGPIOPin)
			for i := 0; i < gpio.MaxGPIOPin; i++ {
				oo[i] = i
			}
		}
	} else {
		oo, err = parseOffsets(args)
		if err != nil {
			return err
		}
	}
	err = gpio.Open()
	if err != nil {
		return err
	}
	defer gpio.Close()
	vv := make([]gpio.Level, len(oo))
	for i, o := range oo {
		pin := gpio.NewPin(o)
		pin.Input()
		v := pin.Read()
		if getOpts.ActiveLow {
			v = !v
		}
		vv[i] = v
	}
	if getOpts.Short {
		printValuesShort(oo, vv)

	} else {
		printValues(oo, vv)
	}
	return nil
}

func printValues(oo []int, vv []gpio.Level) {
	for i, o := range oo {
		fmt.Printf("pin %2d: %t\n", o, vv[i])
	}
}

func printValuesShort(oo []int, vv []gpio.Level) {
	fmt.Printf("%d", level2Int(vv[0]))
	for _, v := range vv[1:] {
		fmt.Printf(" %d", level2Int(v))
	}
	fmt.Println()

}

func parseOffsets(args []string) ([]int, error) {
	oo := []int(nil)
	for _, arg := range args {
		o, err := parseOffset(arg)
		if err != nil {
			return nil, err
		}
		oo = append(oo, int(o))
	}
	return oo, nil
}

func level2Int(l gpio.Level) int {
	if l == gpio.Low {
		return 0
	}
	return 1
}
