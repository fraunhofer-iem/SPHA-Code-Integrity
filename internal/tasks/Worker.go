package tasks

import "sync"

type Worker[T, E any] struct {
	Do func(*T) (*E, error)
}

// helper function for workers that should just pass through their
// input to the output. This is useful to use the workerpool to read
// data from one source and store it to another source (file to database)
// without further modification
func DoNothing[T any](t *T) (*T, error) {
	return t, nil
}

func (w *Worker[T, E]) Run(in <-chan *T, out chan *E, errc chan error, wg *sync.WaitGroup) {
	defer wg.Done()
	for i := range in {
		res, err := w.Do(i)
		if err != nil {
			errc <- err
			continue
		}
		out <- res
	}
}
