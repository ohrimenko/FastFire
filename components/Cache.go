package components

import (
	"errors"
	"log"
	"backnet/config"
	"time"

	"backnet/components/filecache"

	"sync"

	"github.com/dgraph-io/ristretto"
)

type cacheInterface interface {
	Get(interface{}) (interface{}, error)
	Set(interface{}, interface{}, time.Duration) bool
	Del(interface{})
	Close()
	Clear()
}

type cacheStruct struct {
	store cacheInterface
	err   error
	valid bool

	mutex sync.Mutex
}

type ristrettoStruct struct {
	store *ristretto.Cache
	err   error
	valid bool
}

type filecacheStruct struct {
	store *filecache.CacheFile
	err   error
	valid bool
}

func (r *ristrettoStruct) Get(key interface{}) (interface{}, error) {
	if r.valid {
		value, found := r.store.Get(key)
		if found {
			return value, nil
		} else {
			return nil, errors.New("ristrettoStruct not valid")
		}
	} else {
		return nil, errors.New("ristrettoStruct not valid")
	}
}

func (r *ristrettoStruct) Set(key interface{}, value interface{}, ttl time.Duration) bool {
	if r.valid {
		return r.store.SetWithTTL(key, value, 1, ttl)
	} else {
		return false
	}
}

func (r *ristrettoStruct) Del(key interface{}) {
	if r.valid {
		r.store.Del(key)
	}
}

func (r *ristrettoStruct) Close() {
	if r.valid {
		r.store.Close()
	}
}

func (r *ristrettoStruct) Clear() {
	if r.valid {
		r.store.Clear()
	}
}

func (r *filecacheStruct) Get(key interface{}) (interface{}, error) {
	if r.valid {
		value, err := r.store.Get(key)
		if err == nil {
			return value, nil
		} else {
			return nil, err
		}
	} else {
		return nil, errors.New("filecacheStruct not valid")
	}
}

func (r *filecacheStruct) Set(key interface{}, value interface{}, ttl time.Duration) bool {
	if r.valid {
		var interval int64 = 0

		if ttl > time.Second {
			interval = int64(ttl.Seconds())
		}

		return r.store.Set(key, value, interval)
	} else {
		return false
	}
}

func (r *filecacheStruct) Del(key interface{}) {
	if r.valid {
		r.store.Del(key)
	}
}

func (r *filecacheStruct) Close() {
	if r.valid {
	}
}

func (r *filecacheStruct) Clear() {
	if r.valid {
		r.store.Clear()
	}
}

var cacheApp cacheStruct

func (n *cacheStruct) cache() cacheInterface {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if !n.valid {
		if config.Env("CACHE_DRIVER") == "file" {
			r := &filecacheStruct{}

			r.store, r.err = filecache.NewCacheFile("storage/cache")

			if r.err == nil {
				r.valid = true
			}

			n.store = r

			n.err = r.err
		} else if config.Env("CACHE_DRIVER") == "ristretto" {
			r := &ristrettoStruct{}

			r.store, r.err = ristretto.NewCache(&ristretto.Config{
				NumCounters: 1e7,     // number of keys to track frequency of (10M).
				MaxCost:     1 << 30, // maximum cost of cache (1GB).
				BufferItems: 64,      // number of keys per Get buffer.
			})

			if r.err == nil {
				r.valid = true
			}

			n.store = r

			n.err = r.err
		} else {
			log.Fatal(errors.New("CACHE_DRIVER not default"))
		}

		if n.err != nil {
			panic(n.err)
		}

		n.valid = true
	}

	return n.store
}

func Cache(args ...interface{}) interface{} {
	if len(args) == 0 {
		return cacheApp.cache()
	}

	if len(args) == 1 {
		value, err := cacheApp.cache().Get(args[0])
		if err != nil {
			return nil
		} else {
			return value
		}
	}

	if len(args) == 2 {
		switch args[1].(type) {
		case nil:
			cacheApp.cache().Del(args[0])
			return nil
		}
	}

	if len(args) == 2 {
		return cacheApp.cache().Set(args[0], args[1], 1)
	}

	if len(args) == 3 {
		var ttl time.Duration = time.Hour
		switch value := args[2].(type) {
		case time.Duration:
			ttl = value
		case uint:
			ttl = time.Duration(value) * time.Second
		case uint8:
			ttl = time.Duration(value) * time.Second
		case uint16:
			ttl = time.Duration(value) * time.Second
		case uint32:
			ttl = time.Duration(value) * time.Second
		case uint64:
			ttl = time.Duration(value) * time.Second
		case int:
			ttl = time.Duration(value) * time.Second
		case int8:
			ttl = time.Duration(value) * time.Second
		case int16:
			ttl = time.Duration(value) * time.Second
		case int32:
			ttl = time.Duration(value) * time.Second
		case int64:
			ttl = time.Duration(value) * time.Second
		}
		return cacheApp.cache().Set(args[0], args[1], ttl)
	}

	return false
}
