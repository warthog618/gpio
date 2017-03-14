// Copyright © 2017 Kent Gibson <warthog618@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/warthog618/gpio"
	"time"
)

// Watches GPIO 4 (J8 7) and reports when it changes state.
func main() {
	err := gpio.Open()
	if err != nil {
		panic(err)
	}
	defer gpio.Close()
	pin := gpio.NewPin(gpio.J8_7)
	pin.Input()
	pin.PullUp()
	pin.Watch(gpio.EdgeBoth, func(pin *gpio.Pin) {
		fmt.Printf("Pin 4 is %v", pin.Read())
	})
	defer pin.Unwatch()
	// In a real application the main thread would do something useful here.
	// But we'll just run for a minute then exit.
	fmt.Println("Watching Pin 4...")
	time.Sleep(time.Minute)
}
