package awriter

import (
	"fmt"
	"io"
	"sync"

	db "ulambda/debug"
	// np "ulambda/ninep"
	"ulambda/writer"
)

type Writer struct {
	sync.Mutex
	producer *sync.Cond
	consumer *sync.Cond
	wrt      *writer.Writer
	buf      []byte
	len      int
	exit     bool
	err      error
}

func (w *Writer) writer() {
	w.Lock()
	defer w.Unlock()
	for !w.exit {
		for w.len == 0 && !w.exit {
			w.consumer.Wait()
		}
		if w.len > 0 {
			db.DPrintf("AWRITER", "%p writer %v\n", w.wrt, w.len)
			n, err := w.wrt.Write(w.buf[0:w.len])
			if err != nil {
				w.err = err
			} else if n != w.len {
				w.err = io.ErrShortWrite
			}
			w.len = 0
			w.producer.Signal()
		}
	}
	db.DPrintf("AWRITER", "%p writer exit\n", w.wrt)
}

func (w *Writer) Write(p []byte) (int, error) {
	w.Lock()
	defer w.Unlock()

	db.DPrintf("AWRITER", "awrwite %p %v\n", w.wrt, len(p))

	if w.exit {
		return 0, fmt.Errorf("Writer is closed")
	}

	for w.len > 0 && w.err == nil {
		w.producer.Wait()
	}
	if w.err != nil {
		return 0, w.err
	}
	copy(w.buf, p)
	w.len = len(p)
	w.consumer.Signal()
	return len(p), nil
}

func (w *Writer) Close() error {
	w.Lock()
	defer w.Unlock()

	db.DPrintf("AWRITER", "close awrwite %p %v\n", w.wrt, w.exit)
	if w.exit {
		return fmt.Errorf("Writer is closed")
	}

	for w.len > 0 && w.err == nil {
		w.producer.Wait()
	}
	if w.err != nil {
		return w.err
	}
	w.exit = true
	w.consumer.Signal()
	return w.err
	// return w.wrt.Close()
}

func NewWriterSize(wrt *writer.Writer, sz int) *Writer {
	w := &Writer{}
	w.wrt = wrt
	w.producer = sync.NewCond(&w.Mutex)
	w.consumer = sync.NewCond(&w.Mutex)
	w.buf = make([]byte, sz)
	go w.writer()
	return w
}
