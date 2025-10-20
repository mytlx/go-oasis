package pool

import (
	"sync"
	"video-factory/internal/iface"
	"video-factory/pkg/config"
)

type ManagerPool struct {
	Pool   map[string]iface.Manager
	Config *config.AppConfig
	Mutex  sync.RWMutex
}

func NewManagerPool(config *config.AppConfig) *ManagerPool {
	return &ManagerPool{
		Pool:   make(map[string]iface.Manager),
		Config: config,
	}
}

// Get 安全获取 Manager
func (p *ManagerPool) Get(mid string) (iface.Manager, bool) {
	p.Mutex.RLock()
	defer p.Mutex.RUnlock()
	m, ok := p.Pool[mid]
	return m, ok
}

// Add 安全添加 Manager
func (p *ManagerPool) Add(mid string, m iface.Manager) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	p.Pool[mid] = m
}

func (p *ManagerPool) Remove(mid string) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	delete(p.Pool, mid)
}

func (p *ManagerPool) Snapshot() map[string]iface.Manager {
	p.Mutex.RLock()
	defer p.Mutex.RUnlock()

	copyMap := make(map[string]iface.Manager, len(p.Pool))
	for k, v := range p.Pool {
		copyMap[k] = v
	}
	return copyMap
}
