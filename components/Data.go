package components

import (
	"sync"
)

type Data struct {
	mutex  sync.Mutex
	Values map[string]any
}

func NewData() *Data {
	return &Data{
		Values: make(map[string]any),
	}
}

func (d *Data) Set(key string, value any) {
	d.mutex.Lock()
	d.Values[key] = value
	d.mutex.Unlock()
}

func (d *Data) Is(key string) bool {
	if _, ok := d.Values[key]; ok {
		return true
	}

	return false
}

func (d *Data) Get(key string) any {
	if d.Is(key) {
		return d.Values[key]
	}

	return nil
}
