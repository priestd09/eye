package collectors

import (
	"sync"
	"time"
)

func Collect(name string, tags Tags, interval int, fn Collector) chan Data {
	ch := make(chan Data)

	go func(ch chan<- Data) {
		for {
			fields, err := fn()
			if err == nil {
				ch <- Data{
					Name:   name,
					Tags:   tags,
					Fields: fields,
				}
			}
			time.Sleep(time.Second * time.Duration(interval))
		}
	}(ch)

	return ch
}

func Merge(cs []<-chan Data) <-chan Data {
	var wg sync.WaitGroup

	out := make(chan Data)

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c is closed, then calls wg.Done.
	output := func(c <-chan Data) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}

	wg.Add(len(cs))

	for _, c := range cs {
		go output(c)
	}

	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
