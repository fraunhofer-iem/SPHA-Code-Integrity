package tasks

import (
	"log/slog"
	"sync"
)

type BufferedCollector[T any] struct {
	Collect func(t []*T) error
	Buffer  int
}

type BufferedCollectorConfig struct {
	Buffer *int
}

func NewBufferedCollector[T any](
	collect func(t []*T) error,
	config BufferedCollectorConfig,
) *BufferedCollector[T] {

	buffer := 1
	if config.Buffer != nil && *config.Buffer > 1 {
		buffer = *config.Buffer
	}

	return &BufferedCollector[T]{
		Buffer:  buffer,
		Collect: collect,
	}
}

func (w *BufferedCollector[T]) Run(in <-chan *T, errc chan error, wg *sync.WaitGroup) {
	defer wg.Done()

	buffer := []*T{}

	for i := range in {
		if len(buffer) > w.Buffer {
			err := w.Collect(buffer)
			if err != nil {
				errc <- err
			}
			slog.Default().Info("Wrote buffer to target")
			buffer = []*T{}
		}

		buffer = append(buffer, i)
	}

	if len(buffer) > 0 {
		err := w.Collect(buffer)
		if err != nil {
			errc <- err
		}
		slog.Default().Info("Wrote remaining buffer to target")
	}
}
