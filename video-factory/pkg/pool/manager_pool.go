package pool

import (
	"sync"
	"video-factory/internal/manager"
	"video-factory/pkg/config"
)

type ManagerPool struct {
	Pool   map[int64]*manager.Manager
	Config *config.AppConfig
	Mutex  sync.RWMutex
}

func NewManagerPool(config *config.AppConfig) *ManagerPool {
	return &ManagerPool{
		Pool:   make(map[int64]*manager.Manager),
		Config: config,
	}
}

// Get 安全获取 Manager
func (p *ManagerPool) Get(mid int64) (*manager.Manager, bool) {
	p.Mutex.RLock()
	defer p.Mutex.RUnlock()
	m, ok := p.Pool[mid]
	return m, ok
}

// Add 安全添加 Manager
func (p *ManagerPool) Add(mid int64, m *manager.Manager) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	p.Pool[mid] = m
}

func (p *ManagerPool) Remove(mid int64) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	delete(p.Pool, mid)
}

func (p *ManagerPool) Snapshot() map[int64]*manager.Manager {
	p.Mutex.RLock()
	defer p.Mutex.RUnlock()

	copyMap := make(map[int64]*manager.Manager, len(p.Pool))
	for k, v := range p.Pool {
		copyMap[k] = v
	}
	return copyMap
}
