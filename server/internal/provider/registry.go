package provider

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	DefaultBreakerThreshold = 5
	DefaultBreakerTimeout   = 30 * time.Second
	DefaultRateLimit        = 5 // requests per second
)

type Registry struct {
	providers sync.Map // map[string]Provider
	breakers  sync.Map // map[string]*CircuitBreaker
	limiters  sync.Map // map[string]*rate.Limiter
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

func (r *Registry) GetLimiter(name string) *rate.Limiter {
	limiter, _ := r.limiters.LoadOrStore(name, rate.NewLimiter(rate.Limit(DefaultRateLimit), 1))
	return limiter.(*rate.Limiter)
}

func (r *Registry) List() []Provider {
	var list []Provider
	r.providers.Range(func(key, value any) bool {
		list = append(list, value.(Provider))
		return true
	})
	return list
}
