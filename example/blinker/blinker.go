package main

import (
	"fmt"
	"gpio"
	"os"
	"os/signal"
	"time"
)

// This example drives GPIO 4, which is pin J8 7.
// The pin is toggled high and low at 1Hz with a 50% duty cycle.
// Do not run this on a Raspberry Pi which has this pin externally driven.
func main() {
	err := gpio.Open()
	if err != nil {
		panic(err)
	}
	defer gpio.Close()
	pin := gpio.NewPin(gpio.J8_7)
	defer pin.Input()
	pin.Output()
	// capture exit signals to ensure pin is reverted to input on exit.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, os.Kill)
	defer signal.Stop(quit)
	for {
		select {
		case <-time.After(500 * time.Millisecond):
			pin.Toggle()
			fmt.Println("Toggled")
		case <-quit:
			return
		}
	}
}
