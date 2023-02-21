package components

import (
	"sync"

	"github.com/gorilla/mux"
)

type routeStruct struct {
	routes map[string]*mux.Router
	valid  bool

	mutex sync.Mutex
}

var routeApp routeStruct

func (r *routeStruct) route(key string) *mux.Router {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.valid {
		r.routes = make(map[string]*mux.Router)
		r.valid = true
	}

	if _, ok := r.routes[key]; !ok {
		r.routes[key] = mux.NewRouter()
	}

	return r.routes[key]
}

func RouteMux(key string) *mux.Router {
	return routeApp.route(key)
}
