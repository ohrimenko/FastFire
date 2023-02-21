package api

import (
	"backnet/controllers"
)

type ControllerMain struct {
	controllers.Controller
}

func NewControllerMain() ControllerMain {
	controller := ControllerMain{}

	return controller
}
