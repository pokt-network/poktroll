package testdelays

import "sync"

// WaitAll waits for all the provided functions to complete.
// It is used to wait for multiple goroutines to complete before proceeding.
func WaitAll(waitFuncs ...func()) {
	wg := sync.WaitGroup{}
	wg.Add(len(waitFuncs))

	for _, f := range waitFuncs {
		go func(f func()) {
			f()
			wg.Done()
		}(f)
	}

	wg.Wait()
}
