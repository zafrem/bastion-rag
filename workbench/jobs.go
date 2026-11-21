package main

import (
	"context"
	"sort"
	"sync"
	"time"
)

// Job is a dataset run executing in the background, so a large file does not
// tie up a request and the UI can show live progress and offer cancel.
type Job struct {
	ID      string
	Name    string
	Total   int
	started time.Time
	cancel  context.CancelFunc

	mu       sync.Mutex
	state    string // running | done | cancelled
	done     int
	lat      []float64
	outcomes map[string]int
	errors   int
	finished time.Time
	report   *Report
}

type JobStatus struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	State     string         `json:"state"`
	Total     int            `json:"total"`
	Done      int            `json:"done"`
	ElapsedMs float64        `json:"elapsed_ms"`
	PerSec    float64        `json:"per_sec"`
	Latency   LatencyStats   `json:"latency"`
	Outcomes  map[string]int `json:"outcomes"`
	Errors    int            `json:"errors"`
	Report    *Report        `json:"report,omitempty"`
}

func (j *Job) Status() JobStatus {
	j.mu.Lock()
	defer j.mu.Unlock()
	end := time.Now()
	if !j.finished.IsZero() {
		end = j.finished
	}
	el := float64(end.Sub(j.started).Microseconds()) / 1000
	st := JobStatus{ID: j.ID, Name: j.Name, State: j.state, Total: j.Total, Done: j.done, ElapsedMs: el,
		Latency: statsOf(j.lat), Errors: j.errors, Report: j.report, Outcomes: map[string]int{}}
	for k, v := range j.outcomes {
		st.Outcomes[k] = v
	}
	if el > 0 {
		st.PerSec = float64(j.done) / (el / 1000)
	}
	return st
}

type JobManager struct {
	mu   sync.Mutex
	jobs map[string]*Job
	seq  []string
}

func NewJobManager() *JobManager { return &JobManager{jobs: map[string]*Job{}} }

const keepJobs = 20

// Start launches a run and returns at once.
func (m *JobManager) Start(pl Pipeline, name string, cases []Case, opt ScoreOptions) *Job {
	ctx, cancel := context.WithCancel(context.Background())
	j := &Job{ID: uuidV4()[:8], Name: name, Total: CountRuns(cases, opt), started: time.Now(), cancel: cancel,
		state: "running", outcomes: map[string]int{}}
	m.mu.Lock()
	m.jobs[j.ID] = j
	m.seq = append(m.seq, j.ID)
	for len(m.seq) > keepJobs { // drop the oldest
		delete(m.jobs, m.seq[0])
		m.seq = m.seq[1:]
	}
	m.mu.Unlock()

	opt.OnResult = func(cr CaseResult) { // called under the scorer's own lock
		j.mu.Lock()
		j.done++
		j.lat = append(j.lat, cr.LatencyMs)
		key := cr.Outcome
		if cr.Altered {
			key = "delivered_altered"
		}
		j.outcomes[key]++
		if cr.Outcome == "error" {
			j.errors++
		}
		j.mu.Unlock()
	}
	go func() {
		rep := Score(ctx, pl, name, cases, opt)
		j.mu.Lock()
		j.report = &rep
		j.finished = time.Now()
		j.state = "done"
		if ctx.Err() != nil {
			j.state = "cancelled"
		}
		j.mu.Unlock()
		cancel()
	}()
	return j
}

func (m *JobManager) Get(id string) (*Job, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.jobs[id]
	return j, ok
}

func (j *Job) Cancel() { j.cancel() }

// CountRuns is how many pipeline runs a set of cases will make.
func CountRuns(cases []Case, opt ScoreOptions) int {
	want := map[string]bool{}
	for _, c := range opt.Categories {
		want[c] = true
	}
	n := 0
	for _, c := range cases {
		if len(want) > 0 && !want[c.Category] {
			continue
		}
		for _, pn := range c.profileNames() {
			if _, ok := opt.Profiles[pn]; ok {
				n++
			}
		}
	}
	return n * max(opt.Repeat, 1)
}

func profileMap(ps []Profile) map[string]Profile {
	m := map[string]Profile{}
	for _, p := range ps {
		m[p.Name] = p
	}
	return m
}

func nameSet(ps []Profile) map[string]bool {
	m := map[string]bool{}
	for _, p := range ps {
		m[p.Name] = true
	}
	return m
}

func sortedProfileNames(ps []Profile) []string {
	var n []string
	for _, p := range ps {
		n = append(n, p.Name)
	}
	sort.Strings(n)
	return n
}
