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
	modeCmd.Flags().BoolVarP(&modeOpts.All, "all", "a", false, "get all line modes")
	modeCmd.Flags().BoolVarP(&modeOpts.Short, "short", "s", false, "single line output format")
	rootCmd.AddCommand(modeCmd)
}

var (
	modeCmd = &cobra.Command{
		Use:     "mode <pin1>...",
		Short:   "Read the functional mode of a pin or pins",
		PreRunE: premode,
		RunE:    mode,
	}
	modeOpts = struct {
		ActiveLow bool
		Short     bool
		All       bool
	}{}
)

func premode(cmd *cobra.Command, args []string) error {
	if !modeOpts.All {
		return cobra.MinimumNArgs(1)(cmd, args)
	}
	return nil
}

func mode(cmd *cobra.Command, args []string) (err error) {
	var oo []int
	if modeOpts.All {
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
	mm := make([]gpio.Mode, len(oo))
	for i, o := range oo {
		pin := gpio.NewPin(o)
		m := pin.Mode()
		mm[i] = m
	}
	if modeOpts.Short {
		printModesShort(oo, mm)

	} else {
		printModes(oo, mm)
	}
	return nil
}

func printModes(oo []int, mm []gpio.Mode) {
	for i, o := range oo {
		fmt.Printf("pin %2d: %s\n", o, modeNames[mm[i]])
	}
}

func printModesShort(oo []int, mm []gpio.Mode) {
	fmt.Printf("%d", mm[0])
	for _, m := range mm[1:] {
		fmt.Printf(" %d", m)
	}
	fmt.Println()
}

var modeNames = map[gpio.Mode]string{
	gpio.Input:  "input",
	gpio.Output: "output",
	gpio.Alt0:   "alt0",
	gpio.Alt1:   "alt1",
	gpio.Alt2:   "alt2",
	gpio.Alt3:   "alt3",
	gpio.Alt4:   "alt4",
	gpio.Alt5:   "alt5",
}
