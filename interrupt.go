// Interrupt capabilities for DIO Pins.

package gpio

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"syscall"
	"time"
)

const (
	MaxGPIOInterrupt = 54
)

type Edge string

const (
	EdgeNone    Edge = "none"
	EdgeRising  Edge = "rising"
	EdgeFalling Edge = "falling"
	EdgeBoth    Edge = "both"
)

type interrupt struct {
	pin       *Pin
	handler   func(*Pin)
	valueFile *os.File
}

type Watcher struct {
	mu sync.Mutex // Guards the following, and sysfs interactions.
	Fd int
	// Map from pin to value Fd.
	interruptFds map[uint8]int
	// Map from pin Fd to interrupt
	interrupts map[int]*interrupt
}

var defaultWatcher *Watcher

func getDefaultWatcher() *Watcher {
	if defaultWatcher == nil {
		defaultWatcher = NewWatcher()
	}
	return defaultWatcher
}

func NewWatcher() *Watcher {
	Fd, err := syscall.EpollCreate1(0)
	if err != nil {
		panic(fmt.Sprintf("Unable to create epoll: %v", err))
	}
	watcher := &Watcher{
		Fd:           Fd,
		interruptFds: make(map[uint8]int),
		interrupts:   make(map[int]*interrupt)}

	go func() {
		var epollEvents [MaxGPIOInterrupt]syscall.EpollEvent

		for {
			n, err := syscall.EpollWait(watcher.Fd, epollEvents[:], -1)
			if err != nil {
				if err == syscall.EBADF || err == syscall.EINVAL {
					// fd closed so exit
					return
				}
				if err == syscall.EINTR {
					continue
				}
				panic(fmt.Sprintf("EpollWait error: %v", err))
			}
			irqs := make([]*interrupt, 0, n)
			watcher.mu.Lock()
			for _, event := range epollEvents {
				if irq, ok := watcher.interrupts[int(event.Fd)]; ok {
					irqs = append(irqs, irq)
				}
			}
			watcher.mu.Unlock()
			for _, irq := range irqs {
				irq.handler(irq.pin)
			}
		}
	}()
	return watcher
}

func closeInterrupts() {
	watcher := defaultWatcher
	if watcher == nil {
		return
	}
	defaultWatcher = nil
	watcher.Close()
}

// His watch has ended.
func (watcher *Watcher) Close() {
	syscall.Close(watcher.Fd)
	watcher.mu.Lock()
	defer watcher.mu.Unlock()

	for fd := range watcher.interrupts {
		intr := watcher.interrupts[fd]
		intr.valueFile.Close()
		unexport(intr.pin)
	}
	watcher.interrupts = nil
	watcher.interruptFds = nil
}

// Wait for the sysfs GPIO files to become writable.
func waitExported(pin *Pin) error {
	path := fmt.Sprintf("/sys/class/gpio/gpio%v/value", pin.pin)
	if err := waitWriteable(path); err != nil {
		return err
	}
	path = fmt.Sprintf("/sys/class/gpio/gpio%v/edge", pin.pin)
	return waitWriteable(path)
}

func waitWriteable(path string) error {
	try := 0
	for {
		fileInfo, err := os.Stat(path)
		if err == nil && fileInfo.Mode()&0x10 != 0 {
			return nil
		}
		try += 1
		if try > 10 {
			return errors.New("timeout")
		}
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}

func export(pin *Pin) error {
	file, err := os.OpenFile("/sys/class/gpio/export", os.O_WRONLY, os.ModeExclusive)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(strconv.Itoa(int(pin.pin)))
	if e, ok := err.(*os.PathError); ok && e.Err == syscall.EBUSY {
		return nil // EBUSY -> the pin has already been exported
	}
	if err != nil {
		return err
	}
	// wait for pin to be exported on sysfs - can take > 100ms on older Pis
	return waitExported(pin)
}

func unexport(pin *Pin) error {
	file, err := os.OpenFile("/sys/class/gpio/unexport", os.O_WRONLY, os.ModeExclusive)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(strconv.Itoa(int(pin.pin)))
	return err
}

func openValue(pin *Pin) (*os.File, error) {
	path := fmt.Sprintf("/sys/class/gpio/gpio%v/value", pin.pin)
	return os.OpenFile(path, os.O_RDWR, os.ModeExclusive)
}

func setEdge(pin *Pin, edge Edge) error {
	path := fmt.Sprintf("/sys/class/gpio/gpio%v/edge", pin.pin)
	file, err := os.OpenFile(path, os.O_RDWR, os.ModeExclusive)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write([]byte(edge))
	return err
}

// The pin can only be registered once.  Subsequent registers,
// without an Unregister, will return an error.
func (watcher *Watcher) RegisterPin(pin *Pin, edge Edge, handler func(*Pin)) error {
	watcher.mu.Lock()
	defer watcher.mu.Unlock()

	_, ok := watcher.interruptFds[pin.pin]
	if ok {
		return errors.New("watch already exists")
	}
	if err := export(pin); err != nil {
		return err
	}
	if err := setEdge(pin, edge); err != nil {
		return err
	}
	valueFile, err := openValue(pin)
	if err != nil {
		return err
	}
	pinFd := int(valueFile.Fd())

	event := syscall.EpollEvent{Events: syscall.EPOLLET & 0xffffffff}
	if err := syscall.SetNonblock(pinFd, true); err != nil {
		return err
	}
	event.Fd = int32(pinFd)
	if err := syscall.EpollCtl(watcher.Fd, syscall.EPOLL_CTL_ADD, pinFd, &event); err != nil {
		return err
	}
	watcher.interruptFds[pin.pin] = pinFd
	watcher.interrupts[pinFd] = &interrupt{pin: pin, handler: handler, valueFile: valueFile}
	return nil
}

func (watcher *Watcher) UnregisterPin(pin *Pin) {
	watcher.mu.Lock()
	defer watcher.mu.Unlock()

	pinFd, ok := watcher.interruptFds[pin.pin]
	if !ok {
		return
	}
	delete(watcher.interruptFds, pin.pin)
	syscall.EpollCtl(watcher.Fd, syscall.EPOLL_CTL_DEL, pinFd, nil)
	syscall.SetNonblock(pinFd, false)
	intr, ok := watcher.interrupts[pinFd]
	if ok {
		delete(watcher.interrupts, pinFd)
		intr.valueFile.Close()
	}
	unexport(pin)
}

// Watch the pin for changes to level.
// The handler is called immediately, to allow the handler to initialise its state
// with the current level, and then on the specified edges.
// The edge determines which edge to watch.
// There can only be one watcher on the pin at a time.
func (pin *Pin) Watch(edge Edge, handler func(*Pin)) error {
	watcher := getDefaultWatcher()
	return watcher.RegisterPin(pin, edge, handler)
}

// Remove any watch from the pin.
func (pin *Pin) Unwatch() {
	watcher := getDefaultWatcher()
	watcher.UnregisterPin(pin)
}
