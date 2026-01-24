package provider

import (
	"sync"
)

type Registry struct {
	providers sync.Map // map[string]Provider
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) Register(p Provider) {
	r.providers.Store(p.Name(), p)
}

func (r *Registry) Get(name string) Provider {
	val, ok := r.providers.Load(name)
	if !ok { return nil }
	return val.(Provider)
}

func (r *Registry) List() []Provider {
	var list []Provider
	r.providers.Range(func(key, value any) bool {
		list = append(list, value.(Provider))
		return true
	})
	return list
}
