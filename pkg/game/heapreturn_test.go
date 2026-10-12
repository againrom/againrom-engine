package game

import (
	"runtime/debug"
	"runtime/metrics"
	"time"
)

// startHeapReturner returns a test process's freed heap to the operating
// system once the heap it holds is over floor and twice the heap the last
// collection found live. The runtime gives such memory back only slowly, so a
// release shard would otherwise hold its largest test's commit until it
// exits, while the release gate starts jobs by the machine's free commit. It
// changes when memory is returned, never a value a test reads.
func startHeapReturner(every time.Duration, floor uint64) (stop func()) {
	samples := []metrics.Sample{
		{Name: "/memory/classes/heap/objects:bytes"},
		{Name: "/memory/classes/heap/unused:bytes"},
		{Name: "/memory/classes/heap/free:bytes"},
		{Name: "/gc/heap/live:bytes"},
	}
	done, finished := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(finished)
		tick := time.NewTicker(every)
		defer tick.Stop()
		for {
			select {
			case <-done:
				return
			case <-tick.C:
			}
			metrics.Read(samples)
			held := samples[0].Value.Uint64() + samples[1].Value.Uint64() + samples[2].Value.Uint64()
			if held > floor && held > 2*samples[3].Value.Uint64() {
				debug.FreeOSMemory()
			}
		}
	}()
	return func() {
		close(done)
		<-finished
	}
}
