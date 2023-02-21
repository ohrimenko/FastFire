package admin

import (
	"net/http"
	"backnet/components"
	"backnet/controllers"
	"backnet/models"
)

type ControllerAuth struct {
	controllers.Controller
}

func NewControllerAuth() ControllerAuth {
	controller := ControllerAuth{}

	return controller
}

func (сontroller ControllerAuth) Login(w http.ResponseWriter, r *http.Request) {
	request := controllers.NewRequest(w, r)
	defer request.Store()

	if !request.Valid {
		return
	}

	request.View([]string{
		"views/admin/layouts/app.html",
		"views/admin/auth/login.html",
	}, 200, map[string]any{
		"Title":         "Authorize",
		"UrlAdminLogin": components.Route("admin.auth.authorize"),
		"Request":       request,
	})
}

func (сontroller ControllerAuth) Authorize(w http.ResponseWriter, r *http.Request) {
	request := controllers.NewRequest(w, r)
	defer request.Store()

	if !request.Valid {
		return
	}

	r.ParseForm()

	if r.Form.Get("login") != "" && r.Form.Get("password") != "" {
		user := models.NewUser()

		request.DB.Where("`login` = ? ", r.Form.Get("login")).Find(user)

		if user.Valid() {
			if components.CheckPasswordHash(r.Form.Get("password"), user.Password.Get()) {
				request.Sess.Set("AuthUserId", user.Id.Get())

				http.Redirect(w, r, components.Route("admin.index"), http.StatusMovedPermanently)
				return
			} else {
				request.Error("Password", "Не верный пароль")
			}
		} else {
			request.Error("Login", "Пользователь не найден")
		}
	}

	if r.Form.Get("login") == "" {
		request.Error("Login", "Логин не заполнено")
	}

	if r.Form.Get("password") == "" {
		request.Error("Password", "Пароль не заполнено")
	}

	request.Old("Login", r.Form.Get("login"))
	request.Old("Password", r.Form.Get("password"))

	http.Redirect(w, r, components.Route("admin.auth.login"), http.StatusFound)
}

func (сontroller ControllerAuth) Logout(w http.ResponseWriter, r *http.Request) {
	request := controllers.NewRequest(w, r)
	defer request.Store()

	if !request.Valid {
		return
	}

	request.Sess.Delete("AuthUserId")

	http.Redirect(w, r, components.Route("admin.auth.login"), http.StatusFound)
}
