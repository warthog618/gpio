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
	rootCmd.AddCommand(detectCmd)
}

var detectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Identify the GPIO chip",
	Args:  cobra.NoArgs,
	RunE:  detect,
}

func detect(cmd *cobra.Command, args []string) error {
	err := gpio.Open()
	if err != nil {
		return err
	}
	defer gpio.Close()
	switch gpio.Chip() {
	case gpio.BCM2835:
		fmt.Println("bcm2835")
	case gpio.BCM2711:
		fmt.Println("bcm2711")
	default:
		fmt.Println("unknown")
	}
	return nil
}
