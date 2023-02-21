package components

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"backnet/config"
	"strings"
	"sync"
)

type viewStruct struct {
	views map[string]*template.Template
	valid bool

	mutex sync.Mutex
}

var viewApp viewStruct

func (n *viewStruct) view() *viewStruct {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if !n.valid {
		n.views = make(map[string]*template.Template)
		n.valid = true
	}

	return n
}

func View(w http.ResponseWriter, tm []string, status int, data any) error {
	wa := viewApp.view()

	if wa.valid {
		key := strings.Join(tm, ",")

		if _, ok := wa.views[key]; !ok || config.Env("DEBUG") == "true" {
			ts, err := template.ParseFiles(tm...)
			if err != nil {
				if config.Env("DEBUG") == "true" {
					fmt.Println("view template parse error: " + key)
				}
				return errors.New("view template parse error")
			} else {
				wa.views[key] = ts
			}
		}

		if _, ok := wa.views[key]; ok {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(status)
			err := wa.views[key].ExecuteTemplate(w, "base", data)
			if err != nil {
				if config.Env("DEBUG") == "true" {
					fmt.Println(err, key)
				}
				return err
			} else {
				return nil
			}
		} else {
			if config.Env("DEBUG") == "true" {
				fmt.Println("view template error: " + key)
			}
			return errors.New("view template error")
		}
	}
	if config.Env("DEBUG") == "true" {
		fmt.Println("view error")
	}
	return errors.New("view error")
}
