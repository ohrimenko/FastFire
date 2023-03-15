package frontend

import (
	"backnet/components"
	"backnet/controllers"
	"backnet/models"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
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

func (сontroller ControllerMain) MediaData(w http.ResponseWriter, r *http.Request) {
	request := controllers.NewRequest(w, r)
	defer request.Store()

	if !request.Valid {
		return
	}

	manifest := ""
	data, err := ioutil.ReadFile("public/manifest/kyiv.webm.mnf")
	if err == nil {
		components.СonvertAssign(&manifest, data)
	}

	request.View([]string{
		"views/layouts/main.html",
		"views/video/index.html",
	}, 200, map[string]any{
		"Title":    "Video MediaData",
		"Video":    "/static/media/kyiv.webm",
		"Poster":   "/static/media/kyiv.webm.png",
		"Loading":  "/static/img/loading-gif.gif",
		"Manifest": manifest,
	})
}

func (сontroller ControllerMain) MediaDataPreload(w http.ResponseWriter, r *http.Request) {
	request := controllers.NewRequest(w, r)
	defer request.Store()

	if !request.Valid {
		return
	}

	r.ParseForm()

	fileItems := map[string]string{}
	keyItems := map[string]string{}
	rangeItems := map[string]map[string]string{}

	for i, _ := range r.Form {
		splitKey := strings.Split(strings.Replace(i, "]", "", -1), "[")

		if len(splitKey) == 3 || len(splitKey) == 4 {
			if splitKey[0] == "items" {
				if splitKey[2] == "key" {
					keyItems[splitKey[1]] = strings.Join(r.Form[i], "")
				}
				if splitKey[2] == "url" {
					fileItems[splitKey[1]] = regexp.MustCompile("\\?.*$").ReplaceAllString(regexp.MustCompile("^.*media/").ReplaceAllString(strings.Join(r.Form[i], ""), ""), "")
				}
				if splitKey[2] == "range" {
					if _, ok := rangeItems[splitKey[1]]; !ok {
						rangeItems[splitKey[1]] = map[string]string{}
					}

					rangeItems[splitKey[1]][splitKey[3]] = strings.Join(r.Form[i], "")
				}
			}
		}
	}

	for i, _ := range fileItems {
		if _, ok := keyItems[i]; ok {
			if _, ok := rangeItems[i]; ok {
				if _, ok := rangeItems[i]["0"]; ok {
					if _, ok := rangeItems[i]["1"]; ok {
						if components.IsFile("public/media/" + fileItems[i]) {
							f, err := os.Open("public/media/" + fileItems[i])

							if err == nil {
								ns, err := strconv.Atoi(rangeItems[i]["0"])
								if err == nil {
									_, err := f.Seek(int64(ns), 0)

									if err == nil {
										nr, err := strconv.Atoi(rangeItems[i]["1"])
										if err == nil && nr > 0 {
											buf := make([]byte, nr-ns+1)

											_, err := io.ReadAtLeast(f, buf, nr-ns+1)

											if err == nil {
												w.Write(buf)
											} else {
												return
											}
										} else {
											return
										}
									} else {
										return
									}
								} else {
									return
								}
							} else {
								return
							}
						} else {
							return
						}
					} else {
						return
					}
				} else {
					return
				}
			} else {
				return
			}
		} else {
			return
		}
	}
}
