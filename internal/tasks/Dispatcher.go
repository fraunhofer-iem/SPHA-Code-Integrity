package tasks

import (
	"iter"
	"log/slog"
	"math"
	"runtime"
	"sync"
	"time"
)

type Dispatcher[T, E any] struct {
	NoWorker        int
	Worker          Worker[T, E]
	Producer        iter.Seq[T]
	ResultCollector BufferedCollector[E]

	logger        *slog.Logger
	channelBuffer int
	rateLimit     time.Duration
}

type DispatcherConfig struct {
	NoWorker  *int           // optional, defaults to runtime.NumCPU()
	RateLimit *time.Duration // optional, defaults to 0 meaning no rate limit is applied
}

func NewDispatcher[T, E any](Worker Worker[T, E],
	Producer iter.Seq[T],
	ResultCollector BufferedCollector[E],
	config DispatcherConfig) *Dispatcher[T, E] {

	d := Dispatcher[T, E]{
		Worker:          Worker,
		Producer:        Producer,
		ResultCollector: ResultCollector,
	}

	// set number of workers
	var noWorker int
	if config.NoWorker != nil {
		if *config.NoWorker <= 0 {
			noWorker = runtime.NumCPU()
		} else {
			noWorker = *config.NoWorker
		}
	} else {
		noWorker = runtime.NumCPU()
	}
	d.NoWorker = noWorker

	if config.RateLimit != nil && *config.RateLimit > 0 {
		d.rateLimit = *config.RateLimit
	} else {
		d.rateLimit = 0
	}

	// calculate channel buffer size
	channelBuffer := d.ResultCollector.Buffer
	if channelBuffer == math.MaxInt {
		channelBuffer = 100 // limit channel buffer size
	}
	d.channelBuffer = channelBuffer / noWorker

	d.logger = slog.Default()

	return &d
}

func (d *Dispatcher[T, E]) Dispatch() {

	in := make(chan *T)
	out := make(chan *E, d.channelBuffer)
	errc := make(chan error)

	var processWg sync.WaitGroup
	var writeWg sync.WaitGroup

	// setup workers
	for range d.NoWorker {
		processWg.Add(1)
		go d.Worker.Run(in, out, errc, &processWg)
	}
	d.logger.Debug("Started workers", "number workers", d.NoWorker)

	// setup error logging
	go func() {
		for err := range errc {
			d.logger.Error("an error occured in dispatcher", "error", err)
		}
	}()

	// setup writer
	writeWg.Add(1)
	go d.ResultCollector.Run(out, errc, &writeWg)

	// start producer
	if d.rateLimit > 0 {
		throttle := time.Tick(d.rateLimit)
		for e := range d.Producer {
			<-throttle
			in <- &e
		}
	} else {
		for e := range d.Producer {
			in <- &e
		}
	}

	close(in)
	d.logger.Debug("Producer finished")

	processWg.Wait()
	close(out)
	d.logger.Debug("Worker finished")

	writeWg.Wait()
	close(errc)
	d.logger.Debug("Writer finished")
}
