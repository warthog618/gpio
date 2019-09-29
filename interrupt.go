// Copyright © 2017 Kent Gibson <warthog618@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// Interrupt capabilities for DIO Pins.

// +build linux

package gpio

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

const (
	// MaxGPIOInterrupt is the maximum pin number.
	MaxGPIOInterrupt = MaxGPIOPin
)

// Edge represents the change in Pin level that triggers an interrupt.
type Edge string

const (
	// EdgeNone indicates no level transitions will trigger an interrupt
	EdgeNone Edge = "none"

	// EdgeRising indicates an interrupt is triggered when the pin transitions from low to high.
	EdgeRising Edge = "rising"

	// EdgeFalling indicates an interrupt is triggered when the pin transitions from high to low.
	EdgeFalling Edge = "falling"

	// EdgeBoth indicates an interrupt is triggered when the pin changes level.
	EdgeBoth Edge = "both"
)

type interrupt struct {
	pin       *Pin
	handler   func(*Pin)
	valueFile *os.File
}

// Watcher monitors the pins for level transitions that trigger interrupts.
type Watcher struct {
	// Guards the following, and sysfs interactions.
	sync.Mutex

	epfd int

	// Map from pin to value Fd.
	interruptFds map[int]int

	// Map from pin Fd to interrupt
	interrupts map[int]*interrupt

	// closed when the watcher exits.
	doneCh chan struct{}

	// fds of the pipe for the shutdown handshake.
	donefds []int

	// true once the Watcher has been closed.
	closed bool
}

var defaultWatcher *Watcher

func getDefaultWatcher() *Watcher {
	memlock.Lock()
	if defaultWatcher == nil {
		defaultWatcher = NewWatcher()
	}
	memlock.Unlock()
	return defaultWatcher
}

// NewWatcher creates a goroutine that watches Pins for transitions that trigger
// interrupts.
func NewWatcher() *Watcher {
	epfd, err := unix.EpollCreate1(0)
	if err != nil {
		panic(fmt.Sprintf("Unable to create epoll: %v", err))
	}
	p := []int{0, 0}
	err = unix.Pipe2(p, unix.O_CLOEXEC)
	if err != nil {
		panic(fmt.Sprintf("Unable to create pipe: %v", err))
	}
	epv := unix.EpollEvent{Events: unix.EPOLLIN, Fd: int32(p[0])}
	unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, int(p[0]), &epv)
	w := &Watcher{
		epfd:         epfd,
		interruptFds: make(map[int]int),
		interrupts:   make(map[int]*interrupt),
		doneCh:       make(chan struct{}),
		donefds:      p,
	}
	go w.watch()

	return w
}

func (w *Watcher) watch() {
	var epollEvents [MaxGPIOInterrupt]unix.EpollEvent
	defer close(w.doneCh)
	for {
		n, err := unix.EpollWait(w.epfd, epollEvents[:], -1)
		if err != nil {
			if err == unix.EBADF || err == unix.EINVAL {
				// fd closed so exit
				return
			}
			if err == unix.EINTR {
				continue
			}
			panic(fmt.Sprintf("EpollWait error: %v", err))
		}
		for i := 0; i < n; i++ {
			event := epollEvents[i]
			if event.Fd == int32(w.donefds[0]) {
				unix.Close(w.epfd)
				unix.Close(w.donefds[0])
				return
			}
			w.Lock()
			irq, ok := w.interrupts[int(event.Fd)]
			w.Unlock()
			if ok {
				go irq.handler(irq.pin)
			}
		}
	}
}

func closeInterrupts() {
	watcher := defaultWatcher
	if watcher == nil {
		return
	}
	defaultWatcher = nil
	watcher.Close()
}

// Close - His watch has ended.
func (w *Watcher) Close() {
	w.Lock()
	if w.closed {
		w.Unlock()
		return
	}
	w.closed = true
	unix.Write(w.donefds[1], []byte("bye"))
	for fd := range w.interrupts {
		intr := w.interrupts[fd]
		intr.valueFile.Close()
		unexport(intr.pin)
	}
	w.interrupts = nil
	w.interruptFds = nil
	w.Unlock()
	<-w.doneCh
	unix.Close(w.donefds[1])
}

// RegisterPin creates a watch on the given pin.
//
// The pin can only be registered once.  Subsequent registers,
// without an Unregister, will return an error.
func (w *Watcher) RegisterPin(pin *Pin, edge Edge, handler func(*Pin)) (err error) {
	w.Lock()
	defer w.Unlock()

	_, ok := w.interruptFds[pin.pin]
	if ok {
		return ErrBusy
	}
	if err = export(pin); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			unexport(pin)
		}
	}()
	if err = setEdge(pin, edge); err != nil {
		return err
	}
	valueFile, err := openValue(pin)
	if err != nil {
		return err
	}
	pinFd := int(valueFile.Fd())

	event := unix.EpollEvent{Events: unix.EPOLLET & 0xffffffff}
	if err = unix.SetNonblock(pinFd, true); err != nil {
		return err
	}
	event.Fd = int32(pinFd)
	if err := unix.EpollCtl(w.epfd, unix.EPOLL_CTL_ADD, pinFd, &event); err != nil {
		return err
	}
	w.interruptFds[pin.pin] = pinFd
	w.interrupts[pinFd] = &interrupt{pin: pin, handler: handler, valueFile: valueFile}
	return nil
}

// UnregisterPin removes any watch on the Pin.
func (w *Watcher) UnregisterPin(pin *Pin) {
	w.Lock()
	defer w.Unlock()

	pinFd, ok := w.interruptFds[pin.pin]
	if !ok {
		return
	}
	delete(w.interruptFds, pin.pin)
	unix.EpollCtl(w.epfd, unix.EPOLL_CTL_DEL, pinFd, nil)
	unix.SetNonblock(pinFd, false)
	intr, ok := w.interrupts[pinFd]
	if ok {
		delete(w.interrupts, pinFd)
		intr.valueFile.Close()
	}
	unexport(pin)
}

// Watch the pin for changes to level.
//
// The handler is called immediately, to allow the handler to initialise its state
// with the current level, and then on the specified edges.
// The edge determines which edge to watch.
// There can only be one watcher on the pin at a time.
func (p *Pin) Watch(edge Edge, handler func(*Pin)) error {
	watcher := getDefaultWatcher()
	return watcher.RegisterPin(p, edge, handler)
}

// Unwatch removes any watch from the pin.
func (p *Pin) Unwatch() {
	watcher := getDefaultWatcher()
	watcher.UnregisterPin(p)
}

func waitWriteable(path string) error {
	try := 0
	for unix.Access(path, unix.W_OK) != nil {
		try++
		if try > 10 {
			return ErrTimeout
		}
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}

func export(p *Pin) error {
	file, err := os.OpenFile("/sys/class/gpio/export", os.O_WRONLY, os.ModeExclusive)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(strconv.Itoa(int(p.pin)))
	if e, ok := err.(*os.PathError); ok && e.Err == unix.EBUSY {
		return ErrBusy
	}
	if err != nil {
		return err
	}
	// wait for pin to be exported on sysfs - can take > 100ms on older Pis
	return waitExported(p)
}

func openValue(p *Pin) (*os.File, error) {
	path := fmt.Sprintf("/sys/class/gpio/gpio%v/value", p.pin)
	return os.OpenFile(path, os.O_RDWR, os.ModeExclusive)
}

func setEdge(p *Pin, edge Edge) error {
	path := fmt.Sprintf("/sys/class/gpio/gpio%v/edge", p.pin)
	file, err := os.OpenFile(path, os.O_RDWR, os.ModeExclusive)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write([]byte(edge))
	return err
}

func unexport(p *Pin) error {
	file, err := os.OpenFile("/sys/class/gpio/unexport", os.O_WRONLY, os.ModeExclusive)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(strconv.Itoa(int(p.pin)))
	return err
}

// Wait for the sysfs GPIO files to become writable.
func waitExported(p *Pin) error {
	path := fmt.Sprintf("/sys/class/gpio/gpio%v/value", p.pin)
	if err := waitWriteable(path); err != nil {
		return err
	}
	path = fmt.Sprintf("/sys/class/gpio/gpio%v/edge", p.pin)
	return waitWriteable(path)
}

var (
	// ErrTimeout indicates the operation could not be performed within the
	// expected time.
	ErrTimeout = errors.New("timeout")

	// ErrBusy indicates the operation is already active on the pin.
	ErrBusy = errors.New("pin already in use")
)
