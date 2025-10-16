package pool

import (
	"sync"
	"video-factory/manager"
)

type ManagerPool struct {
	Pool  map[string]manager.IManager
	Mutex sync.RWMutex
}

func NewManagerPool() *ManagerPool {
	return &ManagerPool{
		Pool: make(map[string]manager.IManager),
	}
}

// Get 安全获取 Manager
func (p *ManagerPool) Get(mid string) (manager.IManager, bool) {
	p.Mutex.RLock()
	defer p.Mutex.RUnlock()
	m, ok := p.Pool[mid]
	return m, ok
}

// Add 安全添加 Manager
func (p *ManagerPool) Add(mid string, m manager.IManager) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	p.Pool[mid] = m
}

func (p *ManagerPool) Remove(mid string) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	delete(p.Pool, mid)
}
