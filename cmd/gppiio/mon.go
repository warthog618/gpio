// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"
	"github.com/warthog618/gpio"
)

func init() {
	monCmd.Flags().BoolVarP(&monOpts.ActiveLow, "active-low", "l", false, "treat the line state as active low")
	monCmd.Flags().BoolVarP(&monOpts.FallingEdge, "falling-edge", "f", false, "detect only falling edge events")
	monCmd.Flags().BoolVarP(&monOpts.RisingEdge, "rising-edge", "r", false, "detect only rising edge events")
	monCmd.Flags().UintVarP(&monOpts.NumEvents, "num-events", "n", 0, "exit after n edges")
	monCmd.Flags().BoolVarP(&monOpts.Quiet, "quiet", "q", false, "don't display event details")
	monCmd.Flags().BoolVarP(&monOpts.Sync, "sync", "s", false, "display and count the initial sync event")
	monCmd.SetHelpTemplate(monCmd.HelpTemplate() + extendedMonHelp)
	rootCmd.AddCommand(monCmd)
}

var extendedMonHelp = `
By default both rising and falling edge events are detected and reported.
`

var (
	monCmd = &cobra.Command{
		Use:   "mon <offset1>...",
		Short: "Monitor the level of a pin or pins",
		Long:  `Wait for edge events on GPIO pins and print them to standard output.`,
		Args:  cobra.MinimumNArgs(1),
		RunE:  mon,
	}
	monOpts = struct {
		ActiveLow   bool
		RisingEdge  bool
		FallingEdge bool
		Quiet       bool
		Sync        bool
		NumEvents   uint
	}{}
)

type event struct {
	Time  time.Time
	Pin   int
	Level gpio.Level
}

func mon(cmd *cobra.Command, args []string) error {
	if monOpts.RisingEdge && monOpts.FallingEdge {
		return errors.New("can't filter both falling-edge and rising-edge events")
	}
	oo, err := parseOffsets(args)
	if err != nil {
		return err
	}
	err = gpio.Open()
	if err != nil {
		return err
	}
	if monOpts.ActiveLow {
		monOpts.RisingEdge = !monOpts.RisingEdge
		monOpts.FallingEdge = !monOpts.FallingEdge
	}
	var edge gpio.Edge
	switch {
	case monOpts.RisingEdge == monOpts.FallingEdge:
		edge = gpio.EdgeBoth
	case monOpts.RisingEdge:
		edge = gpio.EdgeRising
	case monOpts.FallingEdge:
		edge = gpio.EdgeFalling
	}
	evtchan := make(chan event)
	eh := func(p *gpio.Pin) {
		evt := event{
			Time:  time.Now(),
			Pin:   p.Pin(),
			Level: p.Read(),
		}
		evtchan <- evt
	}
	defer gpio.Close()
	for _, o := range oo {
		pin := gpio.NewPin(o)
		pin.Input()
		pin.Watch(edge, eh)
	}
	monWait(evtchan)
	return nil
}

func monWait(evtchan <-chan event) {
	sigdone := make(chan os.Signal, 1)
	signal.Notify(sigdone, os.Interrupt, os.Kill)
	defer signal.Stop(sigdone)
	count := uint(0)
	pinSynced := make(map[int]bool)
	for {
		select {
		case evt := <-evtchan:
			level := evt.Level
			if monOpts.ActiveLow {
				level = !level
			}
			edge := "rising"
			if level == gpio.Low {
				edge = "falling"
			}
			if monOpts.Sync || pinSynced[evt.Pin] {
				if !monOpts.Quiet {
					fmt.Printf("event:%3d %-7s %s\n", evt.Pin, edge, evt.Time.Format(time.RFC3339Nano))
				}
				count++
				if monOpts.NumEvents > 0 && count >= monOpts.NumEvents {
					return
				}
			}
			pinSynced[evt.Pin] = true
		case <-sigdone:
			return
		}
	}
}
