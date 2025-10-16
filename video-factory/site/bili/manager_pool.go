package bili

import "sync"

type ManagerPool struct {
	Pool  map[string]*Manager
	Mutex sync.RWMutex
}

func NewManagerPool() *ManagerPool {
	return &ManagerPool{
		Pool: make(map[string]*Manager),
	}
}

// Get 安全获取 Manager
func (p *ManagerPool) Get(rid string) (*Manager, bool) {
	p.Mutex.RLock()
	defer p.Mutex.RUnlock()
	manager, ok := p.Pool[rid]
	return manager, ok
}

// Add 安全添加 Manager
func (p *ManagerPool) Add(rid string, manager *Manager) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	p.Pool[rid] = manager
}

func (p *ManagerPool) Remove(rid string) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	delete(p.Pool, rid)
}
