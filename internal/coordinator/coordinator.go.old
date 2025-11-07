package coordinator

import (
	"sync"

	"jordanella.com/pocket-tcg-go/internal/bot"
	"jordanella.com/pocket-tcg-go/internal/cv"
)

// Coordinator manages multiple bot instances
type Coordinator struct {
	instances  map[int]*bot.Bot
	accountMgr *AccountManager
	cvService  *cv.Service // Shared CV service
	config     *bot.Config
	mu         sync.RWMutex
}

// Status represents bot instance status
type Status struct {
	Running bool
	Error   error
}

func NewCoordinator(cfg *bot.Config) (*Coordinator, error) {
	return &Coordinator{
		instances: make(map[int]*bot.Bot),
		config:    cfg,
	}, nil
}

func (c *Coordinator) StartInstance(id int) error {
	// TODO: Implement
	return nil
}

func (c *Coordinator) StopInstance(id int) error {
	// TODO: Implement
	return nil
}

func (c *Coordinator) StopAll() {
	// TODO: Implement
}

func (c *Coordinator) GetInstanceStatus(id int) Status {
	// TODO: Implement
	return Status{}
}
