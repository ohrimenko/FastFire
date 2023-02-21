package controllers

import (
	"errors"
	"net/http"

	"backnet/components"
	"backnet/models"

	"gorm.io/gorm"
)

type Request struct {
	Writer       http.ResponseWriter
	Request      *http.Request
	DB           *gorm.DB
	Sess         *components.Sess
	User         *models.User
	Err          error
	Valid        bool
	ValidDB      bool
	ValidSession bool
	Messages     components.MessagesMap
	Errors       components.MessagesMap
	Olds         components.MessagesMap
}

func NewRequest(w http.ResponseWriter, r *http.Request) *Request {
	req := Request{}

	req.Writer = w
	req.Request = r

	req.Valid = true
	req.ValidDB = false
	req.ValidSession = false

	req.DB, req.Err = components.DB()

	if req.Err != nil {
		req.Valid = false
	} else {
		req.ValidDB = true
	}

	req.User = models.NewUser()

	if req.Request == nil {
		req.Valid = false
		req.Err = errors.New("User Not Found")
	} else {
		s, _ := components.Session(req.Request)

		req.Sess, req.Err = components.NewSess(s)
		if req.Err == nil {
			req.ValidSession = true
			if req.Valid {
				switch value := req.Sess.Get("AuthUserId").(type) {
				case int, uint, int8, uint8, int16, uint16, int32, uint32, int64, uint64:
					req.DB.Find(req.User, "id = ?", value)
				}
			}

			if value, ok := req.Sess.Get("Errors").(components.MessagesMap); ok {
				req.Errors = value
			}

			if value, ok := req.Sess.Get("Messages").(components.MessagesMap); ok {
				req.Messages = value
			}

			if value, ok := req.Sess.Get("Olds").(components.MessagesMap); ok {
				req.Olds = value
			}
		} else {
			req.Valid = false
		}
	}

	if req.Messages == nil {
		req.Messages = components.MessagesMap{}
	}

	if req.Errors == nil {
		req.Errors = components.MessagesMap{}
	}

	if req.Olds == nil {
		req.Olds = components.MessagesMap{}
	}

	if !req.Valid {
		if req.Request != nil {
			req.Err = errors.New("connect not valid")
			Abort500(req.Writer, req.Request)
		}
	}

	return &req
}

func (req *Request) IsAuth() bool {
	if req.Valid {
		if req.User != nil {
			if req.User.Valid() {
				return true
			}
		}
	}

	return false
}

func (req *Request) Auth() *Request {
	if req.Valid {
		if !req.IsAuth() {
			req.Valid = false
		}

		if !req.Valid {
			if req.Request != nil {
				http.Redirect(req.Writer, req.Request, components.Route("main.auth.login"), http.StatusFound)

				req.Err = errors.New("not authorized")
			}
		}
	}

	return req
}

func (req *Request) IsAdmin() bool {
	if req.Valid {
		if req.User != nil {
			if req.User.Valid() {
				if req.User.Type.Get() == 1 {
					return true
				}
			}
		}
	}

	return false
}

func (req *Request) Admin() *Request {
	if req.Valid {
		if !req.IsAdmin() {
			req.Valid = false
		}

		if !req.Valid {
			if req.Request != nil {
				http.Redirect(req.Writer, req.Request, components.Route("admin.auth.login"), http.StatusFound)

				req.Err = errors.New("user not admin")
			}
		}
	}

	return req
}

func (req *Request) Error(key string, value ...any) string {
	if req.Valid {
		if len(value) > 0 {
			req.Errors[key] = &components.Message{}

			for _, v := range value {
				req.Errors[key].Scan(v)
			}
		} else if _, ok := req.Errors[key]; ok {
			req.Errors[key].IsRead = true

			if req.ValidSession {
				req.Sess.Update = true
			}

			return req.Errors[key].Get()
		}
	}

	return ""
}

func (req *Request) Message(key string, value ...any) string {
	if req.Valid {
		if len(value) > 0 {
			req.Messages[key] = &components.Message{}

			for _, v := range value {
				req.Messages[key].Scan(v)
			}

			if req.ValidSession {
				req.Sess.Update = true
			}
		} else if _, ok := req.Messages[key]; ok {
			req.Messages[key].IsRead = true

			if req.ValidSession {
				req.Sess.Update = true
			}

			return req.Messages[key].Get()
		}
	}

	return ""
}

func (req *Request) Old(key string, value ...any) string {
	if req.Valid {
		if len(value) > 0 {
			req.Olds[key] = &components.Message{}

			for _, v := range value {
				req.Olds[key].Scan(v)
			}

			if req.ValidSession {
				req.Sess.Update = true
			}
		} else if _, ok := req.Olds[key]; ok {
			req.Olds[key].IsRead = true

			if req.ValidSession {
				req.Sess.Update = true
			}

			return req.Olds[key].Get()
		}
	}

	return ""
}

func (req *Request) OldOrValue(key string, value ...any) string {
	res := ""

	if req.Valid {
		if _, ok := req.Olds[key]; ok {
			req.Olds[key].IsRead = true
			res = req.Olds[key].String()
		}

		if res == "" && len(value) > 0 {
			for _, v := range value {
				components.СonvertAssign(&res, v)

				if res != "" {
					break
				}
			}
		}
	}

	return res
}

func (req *Request) Store() *Request {
	if req.Valid {
		if req.ValidSession && req.Sess.Update {
			if req.Messages != nil {
				for key, _ := range req.Messages {
					if !req.Messages[key].IsSave() {
						delete(req.Messages, key)
					}
				}

				if len(req.Messages) > 0 {
					req.Sess.Set("Messages", req.Messages)
				} else {
					req.Sess.Delete("Messages")
				}
			}
			if req.Errors != nil {
				for key, _ := range req.Errors {
					if !req.Errors[key].IsSave() {
						delete(req.Errors, key)
					}
				}

				if len(req.Errors) > 0 {
					req.Sess.Set("Errors", req.Errors)
				} else {
					req.Sess.Delete("Errors")
				}
			}
			if req.Olds != nil {
				for key, _ := range req.Olds {
					if !req.Olds[key].IsSave() {
						delete(req.Olds, key)
					}
				}

				if len(req.Olds) > 0 {
					req.Sess.Set("Olds", req.Olds)
				} else {
					req.Sess.Delete("Olds")
				}
			}

			req.Sess.Save(req.Writer, req.Request)
		}
	}

	return req
}

func (req *Request) Session(args ...interface{}) interface{} {
	if len(args) == 1 {
		return req.Sess.Get(args[0])
	}

	if len(args) == 2 {
		switch args[1].(type) {
		case nil:
			req.Sess.Delete(args[0])
			return nil
		}
	}

	if len(args) == 2 {
		req.Sess.Set(args[0], args[1])
	}

	return nil
}

func (req *Request) Cache(args ...interface{}) interface{} {
	return components.Cache(args...)
}

func (req *Request) View(tm []string, status int, data any) error {
	return components.View(req.Writer, tm, status, data)
}
