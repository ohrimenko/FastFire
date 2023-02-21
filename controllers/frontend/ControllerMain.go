package frontend

import (
	"net/http"
	"backnet/controllers"
	"backnet/models"
)

type ControllerMain struct {
	controllers.Controller
}

func NewControllerMain() ControllerMain {
	controller := ControllerMain{}

	return controller
}

func (сontroller ControllerMain) Index(w http.ResponseWriter, r *http.Request) {
	request := controllers.NewRequest(w, r)
	defer request.Store()

	if !request.Valid {
		return
	}

	//vars := mux.Vars(r)
	//r.URL.Query().Get("cmd")

	name := request.Session("name")
	request.Session("name", "john")

	user := models.NewUser()

	request.DB.Find(user, "id = ?", 12501)
	//components.Db.Raw("SELECT * FROM `users` WHERE id = ?", 25005).Scan(&user)

	if !user.Valid() {
		controllers.Abort404(w, r)
		return
	}

	user.Gender.Set("male")
	user.Phone.Scan(nil)
	user.Phone.Set("0974721930")
	user.CoordinateLat.Set(50.450441)
	request.DB.Save(user)

	request.View([]string{
		"views/layouts/main.html",
		"views/index.html",
	}, 200, map[string]any{
		"Title": name,
		"User":  user,
		"Rows": map[string]string{
			"Row1": "Text1",
			"Row2": "Text2",
			"Row3": "Text3",
		},
	})
}
