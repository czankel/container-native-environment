package cli

import (
	"io"
	"sync"

	"github.com/czankel/cne/runtime"
)

// RingBuffer provides a simple ring buffer for buffering output from an output and error writer.
// Regular and error output are intermixed but on a line-level.
type RingBuffer struct {
	mutex      sync.Mutex
	lines      [][]byte
	firstLine  int
	lastLine   int
	isTerminal bool
	out        *lineBuffer
	err        *lineBuffer
}

type lineBuffer struct {
	rngBuf   *RingBuffer
	line     []byte
	capacity int
}

// assumes mutex has been acquired
func (lb *lineBuffer) flush() {

	rb := lb.rngBuf
	rb.lines[rb.lastLine] = append([]byte{}, lb.line...)
	lb.line = lb.line[:0]

	rb.lastLine++
	if rb.lastLine >= len(rb.lines) {
		rb.lastLine = 0
	}
	if rb.lastLine == rb.firstLine {
		rb.firstLine++
		if rb.firstLine >= len(rb.lines) {
			rb.firstLine = 0
		}
	}
}

// Write appends to the current line up to a CR character, in which case the line is copied
// to the ring buffer and cleared. The oldest line is replaced if the capacity of the ring buffer
// is reached.
func (lb *lineBuffer) Write(p []byte) (n int, err error) {

	pLen := len(p)
	rb := lb.rngBuf

	rb.mutex.Lock()
	defer rb.mutex.Unlock()

	for _, c := range p {
		if c == '\n' {
			lb.flush()
		} else if len(lb.line) == lb.capacity-1 {
			// ignore rest of line, change to '...'
			copy(lb.line[lb.capacity-3:], "...")
			break
		} else if len(lb.line) == lb.capacity {
			break
		} else {
			lb.line = append(lb.line, c)
		}
	}
	return pLen, nil
}

// Flush flushes the line buffer and adds it to the ring buffer. It replaces the oldest line if
// the capacity of the ring buffer is reached.
func (lb *lineBuffer) Flush() {

	lb.rngBuf.mutex.Lock()
	defer lb.rngBuf.mutex.Unlock()

	lb.flush()
}

// Read reads a line from the ring buffer into the provided buffer. It returns the number of
// characters read or an error code. If there are no more lines to read, a length of 0 and io.EOF
// error is returned.
func (rb *RingBuffer) Read(p []byte) (int, error) {

	rb.mutex.Lock()
	defer rb.mutex.Unlock()

	if rb.firstLine == rb.lastLine {
		return 0, io.EOF
	}

	copy(p, rb.lines[rb.firstLine])
	rb.firstLine++
	if rb.firstLine >= len(rb.lines) {
		rb.firstLine = 0
	}

	return len(p), nil
}

// Flush flushes all line buffers.
func (rb *RingBuffer) Flush() {

	rb.out.Flush()
	rb.err.Flush()
}

// Reset resets the entire ring buffer (including line buffers)
func (rb *RingBuffer) Reset() {

	rb.mutex.Lock()
	defer rb.mutex.Unlock()

	rb.firstLine = 0
	rb.lastLine = 0

	rb.out.line = rb.out.line[:0]
	rb.err.line = rb.err.line[:0]
}

// NewRingBuffer creates a new ring buffer with a maximum number of lines and line lengths.
func NewRingBuffer(lineCount, lineLength int) *RingBuffer {
	ringBuffer := &RingBuffer{
		lines: make([][]byte, lineCount),
	}
	for i := int(0); i < lineCount; i++ {
		ringBuffer.lines[i] = make([]byte, 0, lineLength)
	}
	outBuffer := &lineBuffer{
		line:     make([]byte, 0, lineLength),
		capacity: lineLength,
		rngBuf:   ringBuffer,
	}
	errBuffer := &lineBuffer{
		line:     make([]byte, 0, lineLength),
		capacity: lineLength,
		rngBuf:   ringBuffer,
	}
	ringBuffer.out = outBuffer
	ringBuffer.err = errBuffer
	return ringBuffer
}

// StreamReader returns a runtime Stream for the RingBuffer for StdOut and StdErr.
func (rb *RingBuffer) StreamWriter() runtime.Stream {
	return runtime.Stream{
		Stdin:    nil,
		Stdout:   rb.out,
		Stderr:   rb.err,
		Terminal: false,
	}
}
