package flow

import (
	"context"
	"github.com/bytepowered/runv"
	"sync"
)

var (
	_ runv.Liveness = new(StateWorker)
)

type StateTask struct {
	Id   string
	Task func() error
}

type StateWorker struct {
	statectx context.Context
	statefun context.CancelFunc
	tasks    []StateTask
	stepwg   sync.WaitGroup
}

func NewStateWorker(ctx context.Context) *StateWorker {
	sc, sf := context.WithCancel(ctx)
	return &StateWorker{
		statectx: sc, statefun: sf,
		tasks: make([]StateTask, 0, 2),
	}
}

func (s *StateWorker) AddWorkTask(id string, task func() error) *StateWorker {
	s.tasks = append(s.tasks, StateTask{Id: id, Task: task})
	return s
}

func (s *StateWorker) Startup(ctx context.Context) error {
	for _, t := range s.tasks {
		go func(task StateTask) {
			defer s.stepwg.Done()
			s.stepwg.Add(1)
			defer Log().Infof("work-task(%s): terminaled", task.Id)
			Log().Infof("work-task(%s): startup", task.Id)
			for {
				select {
				case <-s.statectx.Done():
					Log().Infof("work-task(%s): stop by signal", task.Id)
					return

				default:
					if err := task.Task(); err != nil {
						Log().Errorf("work-task(%s): stop by error: %s", task.Id, err)
						return
					}
				}
			}
		}(t)
	}
	return nil
}

func (s *StateWorker) Shutdown(ctx context.Context) error {
	s.statefun()
	s.stepwg.Wait()
	size := len(s.tasks)
	if size > 0 {
		Log().Infof("tasks(%d): shutdown", size)
	}
	return nil
}
