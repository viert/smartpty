package smartpty

import (
	"github.com/kr/pty"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"sync"
	"syscall"
)

const (
	bufferSize = 4096
)

// ExpressionCallback represents a SmartPTY callback function class
// Use this to react on matches in stdout data. Boolean value returned
// by the function is used to indicate if the match shouldn't be shown
// in stdout
type ExpressionCallback func(data []byte, tty *os.File) []byte

// SmartPTY represents the SmartPTY class
type SmartPTY struct {
	cmd       *exec.Cmd
	callbacks []*cbDescriptor
	signals   chan os.Signal
	tty       *os.File
	finished  bool
	stdinSync *sync.Mutex
}

type cbDescriptor struct {
	expr  *regexp.Regexp
	cb    ExpressionCallback
	count int
}

// Create method creates a new instance of SmartPTY based on exec.Cmd
func Create(cmd *exec.Cmd) *SmartPTY {
	return &SmartPTY{
		cmd,
		make([]*cbDescriptor, 0),
		make(chan os.Signal, 1),
		nil,
		false,
		new(sync.Mutex),
	}
}

// Always method sets a callback which will always
// be called when the given expression occurs in terminal stdout
func (sp *SmartPTY) Always(expression *regexp.Regexp, cb ExpressionCallback) {
	sp.Times(expression, cb, -1)
}

// Once method sets a callback which will be called exactly once
// when the given expression occurs in terminal stdout
func (sp *SmartPTY) Once(expression *regexp.Regexp, cb ExpressionCallback) {
	sp.Times(expression, cb, 1)
}

// Times method sets a callback which will be called
// when the given expression occurs in terminal stdout <times> times max.
// When maximum reactions reached the callback is disabled
func (sp *SmartPTY) Times(expression *regexp.Regexp, cb ExpressionCallback, times int) {
	desc := &cbDescriptor{expression, cb, times}
	sp.callbacks = append(sp.callbacks, desc)
}

// Start starts the process configured
func (sp *SmartPTY) Start() error {
	var err error
	sp.tty, err = pty.Start(sp.cmd)
	if err != nil {
		return err
	}
	go sp.processSignals()
	go sp.processStdout()
	go sp.processStdin()
	return nil
}

func (sp *SmartPTY) processSignals() {
	signal.Notify(sp.signals, syscall.SIGWINCH)
	defer signal.Reset()
	sp.signals <- syscall.SIGWINCH
	for range sp.signals {
		pty.InheritSize(os.Stdin, sp.tty)
	}
}

func (sp *SmartPTY) processStdout() {
	var displayBuffer []byte

	buf := make([]byte, bufferSize)
	shouldCompact := false

	defer sp.tty.Close()

	for !sp.finished {
		n, err := sp.tty.Read(buf)
		if err != nil {
			// EOF
			sp.finished = true
			break
		}

		// copy data for the callback as we'll replace it shortly
		displayBuffer = make([]byte, n)
		copy(displayBuffer, buf[:n])

		// searching for mathes
		for _, cbd := range sp.callbacks {
			if cbd.count == 0 {
				// this callback shouldn't be called anymore
				shouldCompact = true
				continue
			}

			if cbd.expr.Match(displayBuffer) {

				// run the callback
				sp.stdinSync.Lock()
				displayBuffer = cbd.cb(displayBuffer, sp.tty)
				sp.stdinSync.Unlock()

				// decrement callback call counter
				if cbd.count > 0 {
					cbd.count--
				}

			}
		}

		os.Stdout.Write(displayBuffer)

		if shouldCompact {
			dfCallbacks := make([]*cbDescriptor, 0)
			for _, cbd := range sp.callbacks {
				if cbd.count != 0 {
					dfCallbacks = append(dfCallbacks, cbd)
				}
			}
			sp.callbacks = dfCallbacks
		}

	}
	// close the signals channel to shut down processSignals()
	sp.Close()
}

func (sp *SmartPTY) processStdin() {
	// Setup stdin to work in raw mode
	stdinState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		sp.finished = true
		return
	}
	defer terminal.Restore(int(os.Stdin.Fd()), stdinState)

	syscall.SetNonblock(int(os.Stdin.Fd()), true)
	defer syscall.SetNonblock(int(os.Stdin.Fd()), false)

	buf := make([]byte, bufferSize)
	for !sp.finished {
		n, _ := os.Stdin.Read(buf)
		if n > 0 {
			sp.stdinSync.Lock()
			sp.tty.Write(buf[:n])
			sp.stdinSync.Unlock()
		}
	}
}

// Close closes the whole process and shuts down all the goroutines
func (sp *SmartPTY) Close() {
	sp.finished = true
	sp.tty.Close()
	close(sp.signals)
}
