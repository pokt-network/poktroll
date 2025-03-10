package testdelays

import "sync"

// WaitAll waits for all the provided functions to complete.
// It is used to wait for multiple goroutines to complete before proceeding.
func WaitAll(waitFuncs ...func()) {
	if len(waitFuncs) == 0 {
		return
	}

	var wg sync.WaitGroup
	wg.Add(len(waitFuncs))

	for _, fn := range waitFuncs {
		go func(f func()) {
			defer wg.Done()
			f()
		}(fn)
	}

	wg.Wait()
}
