package testchannel

import "fmt"

func DrainChannel[V any](ch <-chan V) (closed bool, err error) {
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return true, nil
			}
			continue
		default:
			return false, fmt.Errorf("observer channel left open")
		}
	}
}
