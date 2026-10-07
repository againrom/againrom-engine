package game

import (
	"fmt"
	"maps"
	"sync"

	"againrom/pkg/ui"
)

// autosaveQueue runs automatic saves off the frame thread, one at a time and in
// submission order. The frame thread captures a Snapshot (a detached value) and
// submits the export and file write; nothing a job reads is written again by
// the frame thread.
//
// There is no resident goroutine: a worker exists only while jobs are pending,
// so an idle queue holds nothing that has to be stopped. wait blocks until the
// queue is idle and is the ordering fence for every later save, load and exit.
type autosaveQueue struct {
	mu      sync.Mutex
	idle    sync.Cond
	pending []func()
	running bool
	// results are finished jobs' outcomes, collected by the frame thread.
	results []autosaveResult
}

type autosaveResult struct {
	timed bool
	// message is the player-visible failure text, empty on success.
	message string
	err     error
}

func newAutosaveQueue() *autosaveQueue {
	q := &autosaveQueue{}
	q.idle.L = &q.mu
	return q
}

// submit queues job. With inline set the job runs before submit returns, on the
// calling goroutine; deterministic headless runs use it so a save is on disk
// before the next frame. A job that panics becomes a failure of its own kind:
// timed says which outcome the player is shown, and failure prefixes the
// message of a non-timed one.
func (q *autosaveQueue) submit(inline, timed bool, failure string, job func() autosaveResult) {
	run := func() {
		var result autosaveResult
		func() {
			defer func() {
				if r := recover(); r != nil {
					err := fmt.Errorf("autosave stopped: %v", r)
					result = autosaveResult{timed: timed, err: err}
					if !timed {
						result.message = failure + err.Error()
					}
				}
			}()
			result = job()
		}()
		q.mu.Lock()
		q.results = append(q.results, result)
		q.mu.Unlock()
	}
	q.mu.Lock()
	if inline && !q.running && len(q.pending) == 0 {
		q.mu.Unlock()
		run()
		return
	}
	q.pending = append(q.pending, run)
	start := !q.running
	q.running = true
	q.mu.Unlock()
	if start {
		go q.work()
	}
}

func (q *autosaveQueue) work() {
	for {
		q.mu.Lock()
		if len(q.pending) == 0 {
			q.running = false
			q.idle.Broadcast()
			q.mu.Unlock()
			return
		}
		next := q.pending[0]
		q.pending = q.pending[1:]
		q.mu.Unlock()
		next()
	}
}

// busy reports whether a job is queued or running.
func (q *autosaveQueue) busy() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.running || len(q.pending) != 0
}

// wait blocks until every submitted job has finished.
func (q *autosaveQueue) wait() {
	q.mu.Lock()
	for q.running || len(q.pending) != 0 {
		q.idle.Wait()
	}
	q.mu.Unlock()
}

// take returns and clears the finished timed or mission-start outcomes, oldest
// first.
func (q *autosaveQueue) take(timed bool) []autosaveResult {
	q.mu.Lock()
	defer q.mu.Unlock()
	var out, keep []autosaveResult
	for _, r := range q.results {
		if r.timed == timed {
			out = append(out, r)
		} else {
			keep = append(keep, r)
		}
	}
	q.results = keep
	return out
}

// detachedExporter returns a front end, and the snapshot to export through it,
// that share nothing the frame thread writes after this call. The export reads
// from the front end the install tables, the campaign definition, the mission's
// decoded map and dead-body art, the world-map registry and the font; every
// other input is in the Snapshot. The tables and the decoded map do not change
// after launch and mission entry.
func (f *FrontEnd) detachedExporter(s Snapshot) (*FrontEnd, Snapshot) {
	s, live := f.detachedMission(s)
	return &FrontEnd{InstallResources: f.InstallResources, Presentation: f.detachedWorldMap(),
		CampaignSession: CampaignSession{live: live}}, s
}

// detachedWorldMap shares the world-map registry only once it is loaded; an
// unloaded one is loaded by the exporting goroutine into its own cache.
func (f *FrontEnd) detachedWorldMap() Presentation {
	var p Presentation
	if a := f.worldMapCache.Value(); f.worldMapCache.Tried() && a != nil {
		p.worldMapCache = resolved(&worldMapAssets{data: a.data, problem: a.problem}, f.worldMapCache.Err())
	}
	return p
}

// detachedMission keeps the mission's immutable map and a copy of its
// dead-body art, and completes the snapshot's ghost template if it has none.
func (f *FrontEnd) detachedMission(s Snapshot) (Snapshot, *mapWorld) {
	if f.live == nil {
		return s, nil
	}
	if s.ghost == nil && f.live.world != nil {
		ghost := f.live.world.Ghost()
		s.ghost = &ghost
	}
	if f.live.mission == nil || f.live.mission.state == nil {
		return s, &mapWorld{}
	}
	ms := f.live.mission.state
	return s, &mapWorld{mission: &missionNotices{state: &Mission{
		Number: ms.Number, Address: ms.Address, Map: ms.Map, DeadArt: maps.Clone(ms.DeadArt)}}}
}

// runsAutosaveInline is true for deterministic headless runs, which write an
// automatic save before the next frame instead of on a worker.
func (f *FrontEnd) runsAutosaveInline() bool { return f.deterministicFrames }

// showsViewer reports whether v is the live mission's viewer.
func (f *FrontEnd) showsViewer(v *ui.Viewer) bool { return f.live != nil && f.live.view == v }
