package queue

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
)

// Queue contains jobs in queue. Only writes to disk to signal
// other processes, not for crash recovery.
// 1. Only one job can be run at the same time.
// 2. Write file to disk to schedule.
// 3. Other process only run the job, named 'TODO'.
type Queue struct {
	mu  sync.RWMutex
	dir string

	todo          *jobToDo
	scheduledJobs []*Job
	completedJobs []*Job

	stopc chan struct{}
	donec chan struct{}
}

// NewQueue returns new Queue. It resets previous queue.
func NewQueue(dir string) (*Queue, error) {
	if err := os.RemoveAll(dir); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0777); err != nil {
		return nil, err
	}

	stopc := make(chan struct{})
	q := &Queue{
		dir: dir,
		todo: &jobToDo{
			path:  filepath.Join(dir, "TODO"),
			txt:   "",
			stopc: stopc,
		},
		stopc: stopc,
		donec: make(chan struct{}),
	}
	go q.run()

	return q, toFile("", q.todo.path)
}

func (q *Queue) run() {
	for {
		select {
		case <-q.stopc:
			glog.Info("stopped")
			close(q.donec)
			return
		default:
		}

		status, err := readJobStatus(q.todo.path)
		if err != nil {
			glog.Error(err)
			continue
		}

		switch status {
		case "": // no job is running
			q.mu.RLock()
			n := len(q.scheduledJobs)
			q.mu.RUnlock()
			if n == 0 {
				select {
				case <-q.stopc:
					glog.Info("stopped")
					close(q.donec)
					return
				case <-time.After(time.Second):
				}
				continue
			}

			q.mu.RLock()
			job := q.scheduledJobs[0]
			q.mu.RUnlock()

			// schedule the first job in the queue
			q.todo.txt = job.txt
			if err = q.todo.fsync(); err != nil {
				glog.Error(err)
				continue
			}

			// wait for fsnotify on 'TODO'
			glog.Infof("scheduled and waiting for %q", job.path)
			select {
			case <-q.todo.fsnotify():
				q.mu.Lock()
				job.txt = q.todo.txt
				job.txt = strings.Replace(job.txt, jobStatus+jobStatusScheduled, jobStatus+jobStatusFinished, 1)
				if len(q.scheduledJobs) > 1 {
					q.scheduledJobs = q.scheduledJobs[1:]
				} else {
					q.scheduledJobs = make([]*Job, 0)
				}
				q.completedJobs = append(q.completedJobs, job)
				close(job.donec)
				q.mu.Unlock()

				q.todo.txt = ""
				if err = q.todo.fsync(); err != nil {
					glog.Error(err)
					continue
				}

			case <-q.stopc:
				glog.Info("stopped")
				close(q.donec)
				return
			}
			glog.Infof("scheduled %q is finished", job.path)

		case jobStatusScheduled, jobStatusFinished: // another job is running
			glog.Fatalf("unexpected job status %q on %q", status, q.todo.path)
		}

		select {
		case <-q.stopc:
			glog.Info("stopped")
			close(q.donec)
			return
		case <-time.After(time.Second):
		}
	}
}

// Schedule schedules a job.
func (q *Queue) Schedule(txt string) *Job {
	q.mu.Lock()
	defer q.mu.Unlock()

	id := ""
	for {
		id = genID()
		if !exist(id) {
			break
		}
	}

	job := &Job{
		path:  filepath.Join(q.dir, id),
		txt:   jobStatus + jobStatusScheduled + "\n\n" + txt,
		donec: make(chan struct{}),
	}
	q.scheduledJobs = append(q.scheduledJobs, job)

	glog.Infof("scheduled %q", job.path)
	return job
}

// Shutdown stops the scheduler.
func (q *Queue) Shutdown() {
	glog.Info("stopping")
	close(q.stopc)
	<-q.donec
}

// Remove removes the job from the scheduler.
func (q *Queue) Remove(j *Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	found := false
	select {
	case <-j.donec:
		glog.Infof("%q is finished, so being removed from queue", j.path)
		jobs := make([]*Job, 0, len(q.completedJobs)-1)
		for _, v := range q.completedJobs {
			if j.path == v.path {
				found = true
				continue
			}
			jobs = append(jobs, v)
		}
		q.completedJobs = jobs

	default:
		glog.Infof("%q is not finished, still being removed from queue")
		jobs := make([]*Job, 0, len(q.scheduledJobs)-1)
		for _, v := range q.scheduledJobs {
			if j.path == v.path {
				found = true
				continue
			}
			jobs = append(jobs, v)
		}
		q.scheduledJobs = jobs
	}

	if !found {
		return fmt.Errorf("%q is not found at completed jobs", j.path)
	}
	return os.RemoveAll(j.path)
}

const (
	jobStatus          = `------Queue STATUS `
	jobStatusScheduled = "SCHEDULED"
	jobStatusFinished  = "FINISHED"
)

// Job represents a task to schedule.
type Job struct {
	path  string
	txt   string
	donec chan struct{}
}

// Notify notifies when the job is done.
// Only notified when the job is already moved to 'Queue' completedJobs.
func (j *Job) Notify() <-chan struct{} {
	return j.donec
}

func readJobStatus(path string) (string, error) {
	f, err := os.OpenFile(path, os.O_RDONLY, 0600)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var status string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		status = scanner.Text()
		status = strings.Replace(status, jobStatus, "", 1)
		break
	}

	return status, scanner.Err()
}

// jobToDo represents a task to run.
type jobToDo struct {
	path  string
	txt   string
	stopc chan struct{}
}

// fsync f-syncs jobToDo to disk.
func (o *jobToDo) fsync() error {
	return toFile(o.txt, o.path)
}

// syncs from disk.
func (o *jobToDo) sync() error {
	bts, err := ioutil.ReadFile(o.path)
	if err != nil {
		return err
	}
	o.txt = string(bts)
	return nil
}

// fsnotify notifies when file events happen.
func (o *jobToDo) fsnotify() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)

	// TODO: use real fsnotify
	prevTxt := o.txt
	for prevTxt == o.txt {
		select {
		case <-o.stopc:
			return ch
		default:
		}
		if err := o.sync(); err != nil {
			glog.Error(err)
			continue
		}
		select {
		case <-o.stopc:
			return ch
		case <-time.After(time.Second):
		}
	}
	return ch
}
