// Copyright © 2017 Kent Gibson <warthog618@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/warthog618/gpio"
)

// Watches GPIO 4 (J8 7) and reports when it changes state.
func main() {
	err := gpio.Open()
	if err != nil {
		panic(err)
	}
	defer gpio.Close()
	pin := gpio.NewPin(gpio.J8p7)
	pin.Input()
	pin.PullUp()

	// capture exit signals to ensure resources are released on exit.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	err = pin.Watch(gpio.EdgeBoth, func(pin *gpio.Pin) {
		fmt.Printf("Pin 4 is %v", pin.Read())
	})
	if err != nil {
		panic(err)
	}
	defer pin.Unwatch()

	// In a real application the main thread would do something useful here.
	// But we'll just run for a minute then exit.
	fmt.Println("Watching Pin 4...")
	select {
	case <-time.After(time.Minute):
	case <-quit:
	}
}
