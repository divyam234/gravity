package provider

import (
	"sync"
	"time"
)

const (
	DefaultBreakerThreshold = 5
	DefaultBreakerTimeout   = 30 * time.Second
)

type Registry struct {
	providers sync.Map // map[string]Provider
	breakers  sync.Map // map[string]*CircuitBreaker
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) Register(p Provider) {
	r.providers.Store(p.Name(), p)
}

func (r *Registry) Get(name string) Provider {
	val, ok := r.providers.Load(name)
	if !ok {
		return nil
	}
	return val.(Provider)
}

func (r *Registry) GetWithBreaker(name string) (Provider, *CircuitBreaker) {
	p := r.Get(name)
	if p == nil {
		return nil, nil
	}

	cb, _ := r.breakers.LoadOrStore(name, NewCircuitBreaker(name, DefaultBreakerThreshold, DefaultBreakerTimeout))
	return p, cb.(*CircuitBreaker)
}

func (r *Registry) List() []Provider {
	var list []Provider
	r.providers.Range(func(key, value any) bool {
		list = append(list, value.(Provider))
		return true
	})
	return list
}
