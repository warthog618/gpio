// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/warthog618/gpio"
)

var rootCmd = &cobra.Command{
	Use:   "gppiio",
	Short: "gppiio is a utility to control Raspberry Pi GPIO pins",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
	Version: version,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func logErr(cmd *cobra.Command, err error) {
	fmt.Fprintf(os.Stderr, "gppiio %s: %s\n", cmd.Name(), err)
}

var pinNames = map[string]int{
	"J8P3":  gpio.J8p3,
	"J8P03": gpio.J8p3,
	"J8P5":  gpio.J8p5,
	"J8P05": gpio.J8p5,
	"J8P7":  gpio.J8p7,
	"J8P07": gpio.J8p7,
	"J8P8":  gpio.J8p8,
	"J8P08": gpio.J8p8,
	"J8P10": gpio.J8p10,
	"J8P11": gpio.J8p11,
	"J8P12": gpio.J8p12,
	"J8P13": gpio.J8p12,
	"J8P15": gpio.J8p15,
	"J8P16": gpio.J8p16,
	"J8P18": gpio.J8p18,
	"J8P19": gpio.J8p19,
	"J8P21": gpio.J8p21,
	"J8P22": gpio.J8p22,
	"J8P23": gpio.J8p23,
	"J8P24": gpio.J8p24,
	"J8P26": gpio.J8p26,
	"J8P27": gpio.J8p27,
	"J8P28": gpio.J8p28,
	"J8P29": gpio.J8p29,
	"J8P31": gpio.J8p31,
	"J8P32": gpio.J8p32,
	"J8P33": gpio.J8p33,
	"J8P35": gpio.J8p35,
	"J8P36": gpio.J8p36,
	"J8P37": gpio.J8p37,
	"J8P38": gpio.J8p38,
	"J8P40": gpio.J8p40,
}

func parseOffset(arg string) (int, error) {
	if o, ok := pinNames[strings.ToUpper(arg)]; ok {
		return o, nil
	}
	o, err := strconv.ParseUint(arg, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("can't parse pin '%s'", arg)
	}
	if o >= gpio.MaxGPIOPin {
		return 0, fmt.Errorf("unknown pin '%d'", o)
	}
	return int(o), nil
}
