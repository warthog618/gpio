// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

package main

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/warthog618/gpio"
)

func init() {
	pullCmd.Flags().BoolVarP(&pullOpts.All, "all", "a", false, "set the pull of all lines")
	pullCmd.Flags().BoolVarP(&pullOpts.Up, "up", "u", false, "pull the line up")
	pullCmd.Flags().BoolVarP(&pullOpts.Down, "down", "d", false, "pull the line down")
	pullCmd.Flags().BoolVarP(&pullOpts.None, "none", "n", false, "disable pull on the line")
	pullCmd.SetHelpTemplate(pullCmd.HelpTemplate() + extendedSetHelp)
	rootCmd.AddCommand(pullCmd)
}

var (
	pullCmd = &cobra.Command{
		Use:     "pull <pin1>...",
		Short:   "Set the pull direction of a pin or pins",
		RunE:    pull,
		Example: "  gppio pull -u J8p15 J8P7",
	}
	pullOpts = struct {
		All  bool
		Up   bool
		Down bool
		None bool
	}{}
)

var extendedPullHelp = `
Pins:
  Pins may be identified by name (J8pXX) or number (0-26).

`

func prepull(cmd *cobra.Command, args []string) error {
	count := 0
	if pullOpts.Up {
		count++
	}
	if pullOpts.Down {
		count++
	}
	if pullOpts.None {
		count++
	}
	if count == 0 {
		return errors.New("must specify pull level [-u|-d|-n]")
	}
	if count > 1 {
		return errors.New("must specify only one pull level")
	}
	if !getOpts.All {
		return cobra.MinimumNArgs(1)(cmd, args)
	}
	return nil
}

func pull(cmd *cobra.Command, args []string) (err error) {
	oo := []int(nil)
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
	var p gpio.Pull
	switch {
	case pullOpts.Up:
		p = gpio.PullUp
	case pullOpts.Down:
		p = gpio.PullDown
	case pullOpts.None:
		p = gpio.PullNone
	}
	for _, o := range oo {
		pin := gpio.NewPin(o)
		pin.SetPull(p)
	}
	return nil
}
