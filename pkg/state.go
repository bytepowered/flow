package flow

import (
	"context"
	"github.com/bytepowered/runv"
	"sync"
)

var (
	_ runv.Liveness = new(StateWorker)
)

type workTask struct {
	Id   string
	Work func() error
}

type stateTask struct {
	Id   string
	Work func(ctx context.Context) error
}

type StateWorker struct {
	statectx   context.Context
	statefun   context.CancelFunc
	workTasks  []workTask
	stateTasks []stateTask
	workwg     sync.WaitGroup
}

func NewStateWorker(ctx context.Context) *StateWorker {
	sc, sf := context.WithCancel(ctx)
	return &StateWorker{
		statectx: sc, statefun: sf,
		workTasks:  make([]workTask, 0, 2),
		stateTasks: make([]stateTask, 0, 2),
	}
}

func (s *StateWorker) AddWorkTask(id string, task func() error) *StateWorker {
	s.workTasks = append(s.workTasks, workTask{Id: id, Work: task})
	return s
}

func (s *StateWorker) AddStateTask(id string, task func(ctx context.Context) error) *StateWorker {
	s.stateTasks = append(s.stateTasks, stateTask{Id: id, Work: task})
	return s
}

func (s *StateWorker) Startup(ctx context.Context) error {
	for _, t := range s.workTasks {
		go s.doWorkTask(t)
	}
	for _, t := range s.stateTasks {
		go s.doStateTask(t)
	}
	return nil
}

func (s *StateWorker) Shutdown(ctx context.Context) error {
	s.statefun()
	s.workwg.Wait()
	size := len(s.workTasks)
	if size > 0 {
		Log().Infof("tasks(%d): shutdown", size)
	}
	return nil
}

func (s *StateWorker) StateContext() context.Context {
	return s.statectx
}

func (s *StateWorker) doStateTask(task stateTask) {
	defer Log().Infof("state-task(%s): terminaled", task.Id)
	Log().Infof("state-task(%s): startup", task.Id)
	if err := task.Work(s.statectx); err != nil {
		Log().Errorf("work-task(%s): stop by error: %s", task.Id, err)
	}
}

func (s *StateWorker) doWorkTask(task workTask) {
	defer s.workwg.Done()
	s.workwg.Add(1)
	defer Log().Infof("work-task(%s): terminaled", task.Id)
	Log().Infof("work-task(%s): startup", task.Id)
	for {
		select {
		case <-s.statectx.Done():
			Log().Infof("work-task(%s): stop by signal", task.Id)
			return

		default:
			if err := task.Work(); err != nil {
				Log().Errorf("work-task(%s): stop by error: %s", task.Id, err)
				return
			}
		}
	}
}
