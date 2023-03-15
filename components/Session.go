package components

import (
	"backnet/config"
	"errors"
	"log"
	"net/http"
	"time"

	"backnet/components/filecache"
	"backnet/components/filecachestore"
	"backnet/components/riststore"

	"sync"

	"github.com/dgraph-io/ristretto"
	"github.com/gorilla/sessions"
	"github.com/wader/gormstore/v2"
)

type storeInterface interface {
	Get(*http.Request, string) (*sessions.Session, error)
}

type storeStruct struct {
	store storeInterface
	err   error
	valid bool

	mutex sync.Mutex
}

type Sess struct {
	Session *sessions.Session
	Valid   bool
	Update  bool
	Init    bool
}

func (s *Sess) Get(key interface{}) interface{} {
	if s.Valid {
		if val, ok := s.Session.Values[key]; ok {
			return val
		}
		return s.Session.Values[key]
	}
	return nil
}

func (s *Sess) Set(key interface{}, val interface{}) {
	if s.Valid {
		s.Session.Values[key] = val
		s.Update = true
	}
}

func (s *Sess) Delete(key interface{}) {
	if s.Valid {
		if _, ok := s.Session.Values[key]; ok {
			delete(s.Session.Values, key)
			s.Update = true
		}
	}
}

func (s *Sess) Save(w http.ResponseWriter, r *http.Request) {
	if s.Valid {
		s.Session.Save(r, w)
	}
}

func NewSess(s *sessions.Session) (*Sess, error) {
	if s != nil {
		return &Sess{
			Session: s,
			Valid:   true,
			Init:    false,
		}, nil
	}

	return nil, errors.New("sess fail create")
}

var storeApp storeStruct

func (n *storeStruct) session() *storeStruct {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if !n.valid {
		if config.Env("SESSION_DRIVER") == "file" {
			switch value := Cache().(type) {
			case *filecacheStruct:
				n.store, n.err = filecachestore.NewFileCacheStore(value.store, []byte(config.Env("SESSION_KEY")))
			default:
				fs, err := filecache.NewCacheFile("storage/cache")

				if err == nil {
					n.store, n.err = filecachestore.NewFileCacheStore(fs, []byte(config.Env("SESSION_KEY")))
				} else {
					n.store = nil
					n.err = errors.New("CACHE_DRIVER file fail create")
				}
			}
		} else if config.Env("SESSION_DRIVER") == "ristretto" {
			switch value := Cache().(type) {
			case *ristrettoStruct:
				n.store, n.err = riststore.NewRistStore(value.store, []byte(config.Env("SESSION_KEY")))
			default:
				rs, err := ristretto.NewCache(&ristretto.Config{
					NumCounters: 1e7,     // number of keys to track frequency of (10M).
					MaxCost:     1 << 30, // maximum cost of cache (1GB).
					BufferItems: 64,      // number of keys per Get buffer.
				})

				if err == nil {
					n.store, n.err = riststore.NewRistStore(rs, []byte(config.Env("SESSION_KEY")))
				} else {
					n.store = nil
					n.err = errors.New("CACHE_DRIVER ristretto fail create")
				}
			}
		} else if config.Env("SESSION_DRIVER") == "DB" {
			db, err := DB()

			if err != nil {
				log.Fatal(err)
			}

			n.store = gormstore.New(db, []byte(config.Env("SESSION_KEY")))

			go n.store.(*gormstore.Store).PeriodicCleanup(1*time.Hour, make(chan struct{}))
		} else {
			n.store = sessions.NewCookieStore([]byte(config.Env("SESSION_KEY")))
		}

		if n.err == nil {
			n.valid = true
		}
	}

	return n
}

func Session(r *http.Request) (*sessions.Session, error) {
	sess := storeApp.session()

	if sess.valid {
		return sess.store.Get(r, "session")
	} else {
		if sess.err != nil {
			return nil, sess.err
		} else {
			return nil, errors.New("session error")
		}
	}
}
