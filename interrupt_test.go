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

	"github.com/stretchr/testify/assert"
)

func waitInterrupt(ch chan int, timeout time.Duration) (int, error) {
	select {
	case v := <-ch:
		return v, nil
	case <-time.After(timeout):
		return 0, errors.New("timeout")
	}
}

func setupIntr(t *testing.T) (pinIn *Pin, pinOut *Pin, watcher *Watcher) {
	assert.Nil(t, Open())
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
	assert.Nil(t, watcher.RegisterPin(pinIn, EdgeRising, func(pin *Pin) {
		count++
		ich <- count
	}))
	v, err := waitInterrupt(ich, 10*time.Millisecond)
	assert.Nil(t, err)
	assert.Equal(t, 1, v)
	_, err = waitInterrupt(ich, 10*time.Millisecond)
	assert.NotNil(t, err, "Spurious interrupt")
}

func TestReregister(t *testing.T) {
	pinIn, pinOut, watcher := setupIntr(t)
	defer teardownIntr(pinIn, pinOut, watcher)
	ich := make(chan int)
	assert.Nil(t, watcher.RegisterPin(pinIn, EdgeRising, func(pin *Pin) {
		ich <- 1
	}))
	v, err := waitInterrupt(ich, 10*time.Millisecond)
	assert.Nil(t, err)
	assert.Equal(t, 1, v)
	assert.NotNil(t, watcher.RegisterPin(pinIn, EdgeRising, func(pin *Pin) {
		ich <- 2
	}), "Reregistration didn't fail.")
	pinOut.High()
	v, err = waitInterrupt(ich, 10*time.Millisecond)
	assert.Nil(t, err)
	assert.Equal(t, 1, v)
}

func TestUnregister(t *testing.T) {
	pinIn, pinOut, watcher := setupIntr(t)
	defer teardownIntr(pinIn, pinOut, watcher)
	ich := make(chan int)
	assert.Nil(t, watcher.RegisterPin(pinIn, EdgeRising, func(pin *Pin) {
		ich <- 1
	}), "Registration failed")
	v, err := waitInterrupt(ich, 10*time.Millisecond)
	assert.Nil(t, err)
	assert.Equal(t, 1, v)
	watcher.UnregisterPin(pinIn)
	pinOut.High()
	_, err = waitInterrupt(ich, 10*time.Millisecond)
	assert.NotNil(t, err)
	// And again just for coverage.
	watcher.UnregisterPin(pinIn)
}

func TestEdgeRising(t *testing.T) {
	pinIn, pinOut, watcher := setupIntr(t)
	defer teardownIntr(pinIn, pinOut, watcher)
	ich := make(chan int)
	assert.Nil(t, watcher.RegisterPin(pinIn, EdgeRising, func(pin *Pin) {
		if pin.Read() == High {
			ich <- 1
		} else {
			ich <- 0
		}
	}))
	v, err := waitInterrupt(ich, 10*time.Millisecond)
	assert.Nil(t, err)
	assert.Equal(t, 0, v)
	// Can take a while for the init to be applied before it starts triggering
	// interrupts, so wait a bit...
	time.Sleep(time.Millisecond)
	for i := 0; i < 10; i++ {
		pinOut.High()
		v, err := waitInterrupt(ich, 10*time.Millisecond)
		if err != nil {
			t.Error("Missed high at", i)
		} else if v == 0 {
			t.Error("Triggered while low at", i)
		}
		pinOut.Low()
		_, err = waitInterrupt(ich, 10*time.Millisecond)
		if err == nil {
			t.Error("Spurious or delayed trigger at", i)
		}
	}
}

func TestEdgeFalling(t *testing.T) {
	pinIn, pinOut, watcher := setupIntr(t)
	defer teardownIntr(pinIn, pinOut, watcher)
	ich := make(chan int)
	assert.Nil(t, watcher.RegisterPin(pinIn, EdgeFalling, func(pin *Pin) {
		if pin.Read() == High {
			ich <- 1
		} else {
			ich <- 0
		}
	}))
	v, err := waitInterrupt(ich, 10*time.Millisecond)
	assert.Nil(t, err)
	assert.Equal(t, 0, v)
	for i := 0; i < 10; i++ {
		pinOut.High()
		_, err := waitInterrupt(ich, 10*time.Millisecond)
		if err == nil {
			t.Error("Spurious or delayed trigger at", i)
		}
		pinOut.Low()
		v, err = waitInterrupt(ich, 10*time.Millisecond)
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
	assert.Nil(t, watcher.RegisterPin(pinIn, EdgeBoth, func(pin *Pin) {
		if pin.Read() == High {
			ich <- 1
		} else {
			ich <- 0
		}
	}))
	v, err := waitInterrupt(ich, 10*time.Millisecond)
	assert.Nil(t, err)
	assert.Equal(t, 0, v)
	for i := 0; i < 10; i++ {
		pinOut.High()
		v, err := waitInterrupt(ich, 10*time.Millisecond)
		if err != nil {
			t.Error("Missed high at", i)
		} else if v == 0 {
			t.Error("Triggered while low at", i)
		}
		pinOut.Low()
		v, err = waitInterrupt(ich, 10*time.Millisecond)
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
	assert.Nil(t, watcher.RegisterPin(pinIn, EdgeNone, func(pin *Pin) {
		if pin.Read() == High {
			ich <- 1
		} else {
			ich <- 0
		}
	}))
	v, err := waitInterrupt(ich, 10*time.Millisecond)
	assert.Nil(t, err)
	assert.Equal(t, 0, v)
	for i := 0; i < 10; i++ {
		pinOut.High()
		v, err := waitInterrupt(ich, 10*time.Millisecond)
		if err == nil {
			t.Error("Spurious or delayed trigger at", i, v)
		}
		pinOut.Low()
		v, err = waitInterrupt(ich, 10*time.Millisecond)
		if err == nil {
			t.Error("Spurious or delayed trigger at", i, v)
		}
	}
}

func TestUnexportedEdge(t *testing.T) {
	pinIn, pinOut, watcher := setupIntr(t)
	assert.NotNil(t, setEdge(pinIn, EdgeNone))
	defer teardownIntr(pinIn, pinOut, watcher)
}

func TestCloseInterrupts(t *testing.T) {
	pinIn, pinOut, watcher := setupIntr(t)
	defer teardownIntr(pinIn, pinOut, watcher)
	ich := make(chan int)
	assert.Nil(t, watcher.RegisterPin(pinIn, EdgeNone, func(pin *Pin) {
		if pin.Read() == High {
			ich <- 1
		} else {
			ich <- 0
		}
	}))
	closeInterrupts()
	_, err := waitInterrupt(ich, 10*time.Millisecond)
	assert.NotNil(t, err, "Spurious interrupt during close")
	pinOut.High()
	_, err = waitInterrupt(ich, 10*time.Millisecond)
	assert.NotNil(t, err, "Interrupts still active after close")
}

func TestWatchExists(t *testing.T) {
	assert.Nil(t, Open())
	defer Close()
	pinIn := NewPin(J8p15)
	pinIn.SetMode(Input)
	count := 0
	assert.Nil(t, pinIn.Watch(EdgeFalling, func(pin *Pin) {
		count++
	}))
	assert.NotNil(t, pinIn.Watch(EdgeFalling, func(pin *Pin) {
		count++
	}))
	time.Sleep(2 * time.Millisecond)
	if count != 1 {
		t.Error("Second handler called")
	}
}

// Looped tests require a jumper across Raspberry Pi J8 pins 15 and 16.
// This is just a smoke test for the Watch and Unwatch methods.
func TestWatchLooped(t *testing.T) {
	assert.Nil(t, Open())
	defer Close()
	pinIn := NewPin(J8p15)
	pinOut := NewPin(J8p16)
	pinIn.SetMode(Input)
	defer pinOut.SetMode(Input)
	pinOut.Write(Low)
	pinOut.SetMode(Output)
	mode := pinOut.Mode()
	assert.Equal(t, Output, mode)
	called := false
	assert.Nil(t, pinIn.Watch(EdgeFalling, func(pin *Pin) {
		called = true
	}))
	time.Sleep(2 * time.Millisecond)
	assert.True(t, called)
	called = false
	pinOut.High()
	time.Sleep(2 * time.Millisecond)
	assert.False(t, called)
	pinOut.Low()
	time.Sleep(2 * time.Millisecond)
	assert.True(t, called)
	pinIn.Unwatch()
	called = false
	pinOut.High()
	pinOut.Low()
	time.Sleep(2 * time.Millisecond)
	assert.False(t, called)
}

// This provides a coarse estimate of the interrupt latency,
// i.e. the time between an interrupt being triggered and handled.
// There is some overhead in there due to the handshaking via a channel etc...
// so this provides an upper bound.
func BenchmarkInterruptLatency(b *testing.B) {
	assert.Nil(b, Open())
	defer Close()
	pinIn := NewPin(J8p15)
	pinOut := NewPin(J8p16)
	pinIn.SetMode(Input)
	defer pinOut.SetMode(Input)
	pinOut.Write(Low)
	pinOut.SetMode(Output)
	mode := pinOut.Mode()
	assert.Equal(b, Output, mode)
	ich := make(chan int)
	assert.Nil(b, pinIn.Watch(EdgeBoth, func(pin *Pin) {
		ich <- 1
	}))
	defer pinIn.Unwatch()
	for i := 0; i < b.N; i++ {
		pinOut.Toggle()
		<-ich
	}
}
