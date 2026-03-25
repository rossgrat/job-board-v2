package scheduler

import "github.com/rossgrat/job-board-v2/plugin/runner"

type Scheduler struct{}

func New() *Scheduler {
	return &Scheduler{}
}

func (s *Scheduler) NewRunner() runner.RunnerFunc {

}
