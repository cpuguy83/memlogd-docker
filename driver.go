package main

import (
	"context"
	"io"
	"log"
	"sync"
	"syscall"

	"github.com/docker/docker/api/types/plugins/logdriver"
	"github.com/pkg/errors"
	"github.com/tonistiigi/fifo"
)

func newDriver() *driver {
	return &driver{
		writers: make(map[string]*logger),
	}
}

type driver struct {
	mu      sync.Mutex
	writers map[string]*logger
}

type logger struct {
	stdout, stderr io.WriteCloser
	stream         io.ReadCloser
}

func (l *logger) Close() error {
	l.stream.Close()
	l.stdout.Close()
	l.stderr.Close()
	return nil
}

func (d *driver) StartLogging(file, id string) error {
	log.Println("start logging: ", file)
	stdout, stderr, err := newMemlogStream(memlogSocket, id)
	if err != nil {
		return err
	}

	f, err := fifo.OpenFifo(context.Background(), file, syscall.O_RDONLY, 0700)
	if err != nil {
		return errors.Wrapf(err, "error opening logger fifo at %s", file)
	}

	d.mu.Lock()
	l := &logger{stdout, stderr, f}
	d.writers[file] = l
	d.mu.Unlock()

	go proxyLogs(l)
	return nil
}

func (d *driver) StopLogging(file string) error {
	log.Println("stop logging: ", file)
	d.mu.Lock()
	var err error
	if l, exists := d.writers[file]; exists {
		err = l.Close()
		delete(d.writers, file)
	}
	d.mu.Unlock()

	return err
}

func proxyLogs(l *logger) {
	dec := logdriver.NewLogEntryDecoder(l.stream)
	var e logdriver.LogEntry
	for {
		if err := dec.Decode(&e); err != nil {
			// TODO: DO SOMETHING
			if err != io.EOF {
				log.Fatal(err)
			}
			return
		}

		var err error
		switch e.Source {
		case "stdout":
			_, err = l.stdout.Write(e.Line)
		case "stderr":
			_, err = l.stderr.Write(e.Line)
		default:
			log.Println("got unexpected log source: %s", e.Source)
		}
		if err != nil {
			// TODO: Try to re-establish a conn?
			log.Println(err)
			return
		}
	}
}
