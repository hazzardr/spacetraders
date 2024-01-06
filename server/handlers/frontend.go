package handlers

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

type Frontend struct {
	AgentsHandler *AgentsAPIHandler
	ShipsHandler  *ShipsAPIHandler
}

func (f *Frontend) LandingPage(c echo.Context) error {
	return c.Render(http.StatusOK, "hello", "text!!")
}
