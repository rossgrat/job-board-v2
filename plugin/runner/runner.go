package runner

import (
	"context"
)

type RunnerFunc func(context.Context) func() error

type Runner struct {
}

func (r *Runner) Run() {

}
