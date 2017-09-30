// Copyright © 2017 Kent Gibson <warthog618@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

//
// Test suite for interrupt module.
//
// Tests use Raspberry Pi J8 pins 15 and 16 which must be jumpered together.
//
package gpio

import (
	"errors"
	"testing"
	"time"
)

func waitInterrupt(ch chan int, timeout time.Duration) (int, error) {
	expired := make(chan bool)
	go func() {
		time.Sleep(timeout)
		close(expired)
	}()
	select {
	case v := <-ch:
		return v, nil
	case <-expired:
		return 0, errors.New("timeout")
	}
}

func setupIntr(t *testing.T) (pinIn *Pin, pinOut *Pin, watcher *Watcher) {
	if err := Open(); err != nil {
		t.Fatal("Open returned error", err)
	}
	pinIn = NewPin(J8p15)
	pinOut = NewPin(J8p16)
	watcher = getDefaultWatcher()
	pinIn.SetMode(Input)
	pinOut.Write(Low)
	pinOut.SetMode(Output)
	return
}

func teardownIntr(pinIn *Pin, pinOut *Pin, watcher *Watcher) {
	pinOut.SetMode(Input)
	watcher.UnregisterPin(pinIn)
	Close()
}

func TestRegister(t *testing.T) {
	pinIn, pinOut, watcher := setupIntr(t)
	defer teardownIntr(pinIn, pinOut, watcher)
	ich := make(chan int)
	count := 0
	err := watcher.RegisterPin(pinIn, EdgeRising, func(pin *Pin) {
		count++
		ich <- count
	})
	if err != nil {
		t.Fatal("Registration failed", err)
	}
	v, err := waitInterrupt(ich, time.Millisecond)
	if err != nil {
		t.Error("No initial interrupt")
	}
	if v != 1 {
		t.Error("Incorrect count", v)
	}
	_, err = waitInterrupt(ich, time.Millisecond)
	if err == nil {
		t.Error("Spurious interrupt")
	}
}

func TestReregister(t *testing.T) {
	pinIn, pinOut, watcher := setupIntr(t)
	defer teardownIntr(pinIn, pinOut, watcher)
	ich := make(chan int)
	if err := watcher.RegisterPin(pinIn, EdgeRising, func(pin *Pin) {
		ich <- 1
	}); err != nil {
		t.Fatal("Registration failed", err)
	}
	v, err := waitInterrupt(ich, time.Millisecond)
	if err != nil {
		t.Error("No initial interrupt")
	}
	if v != 1 {
		t.Error("Incorrect count", v)
	}
	if err = watcher.RegisterPin(pinIn, EdgeRising, func(pin *Pin) {
		ich <- 2
	}); err == nil {
		t.Fatal("Reregistration didn't fail.")
	}
	pinOut.High()
	v, err = waitInterrupt(ich, time.Millisecond)
	switch {
	case err != nil:
		t.Error("Didn't call handler")
	case v != 1:
		t.Error("Called new handler")
	}
}

func TestUnregister(t *testing.T) {
	pinIn, pinOut, watcher := setupIntr(t)
	defer teardownIntr(pinIn, pinOut, watcher)
	ich := make(chan int)
	err := watcher.RegisterPin(pinIn, EdgeRising, func(pin *Pin) {
		ich <- 1
	})
	if err != nil {
		t.Fatal("Registration failed", err)
	}
	v, err := waitInterrupt(ich, time.Millisecond)
	if err != nil {
		t.Error("No initial interrupt")
	}
	if v != 1 {
		t.Error("Incorrect count", v)
	}
	watcher.UnregisterPin(pinIn)
	pinOut.High()
	_, err = waitInterrupt(ich, time.Millisecond)
	if err == nil {
		t.Error("Called old handler")
	}
	// And again just for coverage.
	watcher.UnregisterPin(pinIn)
}

func TestEdgeRising(t *testing.T) {
	pinIn, pinOut, watcher := setupIntr(t)
	defer teardownIntr(pinIn, pinOut, watcher)
	ich := make(chan int)
	err := watcher.RegisterPin(pinIn, EdgeRising, func(pin *Pin) {
		if pin.Read() == High {
			ich <- 1
		} else {
			ich <- 0
		}
	})
	if err != nil {
		t.Fatal("Registration failed", err)
	}
	v, err := waitInterrupt(ich, time.Millisecond)
	if err != nil {
		t.Error("No initial interrupt")
	}
	if v != 0 {
		t.Error("Incorrect initial value", v)
	}
	// Can take a while for the init to be applied before it starts triggering
	// interrupts, so wait a bit...
	time.Sleep(time.Millisecond)
	for i := 0; i < 10; i++ {
		pinOut.High()
		v, err := waitInterrupt(ich, time.Millisecond)
		if err != nil {
			t.Error("Missed high at", i)
		} else if v == 0 {
			t.Error("Triggered while low at", i)
		}
		pinOut.Low()
		_, err = waitInterrupt(ich, time.Millisecond)
		if err == nil {
			t.Error("Spurious or delayed trigger at", i)
		}
	}
}

func TestEdgeFalling(t *testing.T) {
	pinIn, pinOut, watcher := setupIntr(t)
	defer teardownIntr(pinIn, pinOut, watcher)
	ich := make(chan int)
	err := watcher.RegisterPin(pinIn, EdgeFalling, func(pin *Pin) {
		if pin.Read() == High {
			ich <- 1
		} else {
			ich <- 0
		}
	})
	if err != nil {
		t.Fatal("Registration failed", err)
	}
	v, err := waitInterrupt(ich, time.Millisecond)
	if err != nil {
		t.Error("No initial interrupt")
	}
	if v != 0 {
		t.Error("Incorrect initial value", v)
	}
	for i := 0; i < 10; i++ {
		pinOut.High()
		_, err := waitInterrupt(ich, time.Millisecond)
		if err == nil {
			t.Error("Spurious or delayed trigger at", i)
		}
		pinOut.Low()
		v, err = waitInterrupt(ich, time.Millisecond)
		if err != nil {
			t.Error("Missed low at", i)
		} else if v == 1 {
			t.Error("Triggered while low at", i)
		}
	}
}

func TestEdgeBoth(t *testing.T) {
	pinIn, pinOut, watcher := setupIntr(t)
	defer teardownIntr(pinIn, pinOut, watcher)
	ich := make(chan int)
	err := watcher.RegisterPin(pinIn, EdgeBoth, func(pin *Pin) {
		if pin.Read() == High {
			ich <- 1
		} else {
			ich <- 0
		}
	})
	if err != nil {
		t.Fatal("Registration failed", err)
	}
	v, err := waitInterrupt(ich, time.Millisecond)
	if err != nil {
		t.Error("No initial interrupt")
	}
	if v != 0 {
		t.Error("Incorrect initial value", v)
	}
	for i := 0; i < 10; i++ {
		pinOut.High()
		v, err := waitInterrupt(ich, time.Millisecond)
		if err != nil {
			t.Error("Missed high at", i)
		} else if v == 0 {
			t.Error("Triggered while low at", i)
		}
		pinOut.Low()
		v, err = waitInterrupt(ich, time.Millisecond)
		if err != nil {
			t.Error("Missed low at", i)
		} else if v == 1 {
			t.Error("Triggered while high at", i)
		}
	}
}

func TestEdgeNone(t *testing.T) {
	pinIn, pinOut, watcher := setupIntr(t)
	defer teardownIntr(pinIn, pinOut, watcher)
	ich := make(chan int)
	err := watcher.RegisterPin(pinIn, EdgeNone, func(pin *Pin) {
		if pin.Read() == High {
			ich <- 1
		} else {
			ich <- 0
		}
	})
	if err != nil {
		t.Fatal("Registration failed", err)
	}
	v, err := waitInterrupt(ich, time.Millisecond)
	if err != nil {
		t.Error("No initial interrupt")
	}
	if v != 0 {
		t.Error("Incorrect initial value", v)
	}
	for i := 0; i < 10; i++ {
		pinOut.High()
		v, err := waitInterrupt(ich, time.Millisecond)
		if err == nil {
			t.Error("Spurious or delayed trigger at", i, v)
		}
		pinOut.Low()
		v, err = waitInterrupt(ich, time.Millisecond)
		if err == nil {
			t.Error("Spurious or delayed trigger at", i, v)
		}
	}
}

func TestUnexportedEdge(t *testing.T) {
	pinIn, pinOut, watcher := setupIntr(t)
	err := setEdge(pinIn, EdgeNone)
	if err == nil {
		t.Error("Edge should fail unless pin exported first.")
	}
	defer teardownIntr(pinIn, pinOut, watcher)
}

func TestCloseInterrupts(t *testing.T) {
	pinIn, pinOut, watcher := setupIntr(t)
	defer teardownIntr(pinIn, pinOut, watcher)
	ich := make(chan int)
	err := watcher.RegisterPin(pinIn, EdgeNone, func(pin *Pin) {
		if pin.Read() == High {
			ich <- 1
		} else {
			ich <- 0
		}
	})
	if err != nil {
		t.Fatal("Registration failed", err)
	}
	closeInterrupts()
	v, err := waitInterrupt(ich, time.Millisecond)
	if err == nil {
		t.Error("Spurious interrupt during close", v)
	}
	pinOut.High()
	v, err = waitInterrupt(ich, time.Millisecond)
	if err == nil {
		t.Error("Interrupts still active after close", v)
	}
}

func TestWatchExists(t *testing.T) {
	if err := Open(); err != nil {
		t.Fatal("Open returned error", err)
	}
	defer Close()
	pinIn := NewPin(J8p15)
	pinIn.SetMode(Input)
	count := 0
	if err := pinIn.Watch(EdgeFalling, func(pin *Pin) {
		count++
	}); err != nil {
		t.Fatal("Watch returned error", err)
	}
	if err := pinIn.Watch(EdgeFalling, func(pin *Pin) {
		count++
	}); err == nil {
		t.Error("Watch didn't return error")
	}
	time.Sleep(2 * time.Millisecond)
	if count != 1 {
		t.Error("Second handler called")
	}
}

// Looped tests require a jumper across Raspberry Pi J8 pins 15 and 16.
// This is just a smoke test for the Watch and Unwatch methods.
func TestWatchLooped(t *testing.T) {
	if err := Open(); err != nil {
		t.Fatal("Open returned error", err)
	}
	defer Close()
	pinIn := NewPin(J8p15)
	pinOut := NewPin(J8p16)
	pinIn.SetMode(Input)
	defer pinOut.SetMode(Input)
	pinOut.Write(Low)
	pinOut.SetMode(Output)
	mode := pinOut.Mode()
	if mode != Output {
		t.Fatal("Failed to set output")
	}
	called := false
	if err := pinIn.Watch(EdgeFalling, func(pin *Pin) {
		called = true
	}); err != nil {
		t.Fatal("Watch returned error", err)
	}
	time.Sleep(2 * time.Millisecond)
	if called == false {
		t.Error("Watcher not called with initial value.")
	}
	called = false
	pinOut.High()
	time.Sleep(2 * time.Millisecond)
	if called {
		t.Error("Spurious Watcher called.")
		called = false
	}
	pinOut.Low()
	time.Sleep(2 * time.Millisecond)
	if called == false {
		t.Error("Watcher not called.")
	}
	pinIn.Unwatch()
	called = false
	pinOut.High()
	pinOut.Low()
	time.Sleep(2 * time.Millisecond)
	if called != false {
		t.Error("Watcher called after unwatch.")
	}
}

// This provides a coarse estimate of the interrupt latency,
// i.e. the time between an interrupt being triggered and handled.
// There is some overhead in there due to the handshaking via a channel etc...
// so this provides an upper bound.
func BenchmarkInterruptLatency(b *testing.B) {
	if err := Open(); err != nil {
		b.Fatal("Open returned error", err)
	}
	defer Close()
	pinIn := NewPin(J8p15)
	pinOut := NewPin(J8p16)
	pinIn.SetMode(Input)
	defer pinOut.SetMode(Input)
	pinOut.Write(Low)
	pinOut.SetMode(Output)
	mode := pinOut.Mode()
	if mode != Output {
		b.Fatal("Failed to set output")
	}
	ich := make(chan int)
	err := pinIn.Watch(EdgeBoth, func(pin *Pin) {
		ich <- 1
	})
	if err != nil {
		b.Fatal("Registration failed", err)
	}
	defer pinIn.Unwatch()
	for i := 0; i < b.N; i++ {
		pinOut.Toggle()
		waitInterrupt(ich, 5*time.Millisecond)
	}
}
