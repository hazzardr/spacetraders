package handlers

import (
	spaceTraders "github.com/hazzardr/spacetraders/generated/spacetraders"
	"github.com/hazzardr/spacetraders/server"
	"github.com/labstack/echo/v4"
)

type ShipsHandler struct {
	SpaceTraderClient *spaceTraders.ClientWithResponses
	DBOperations      *server.DatabaseOperations
}

func (h ShipsHandler) GetShipShipId(ctx echo.Context, id int) error {
	//TODO: Implement
	panic("implement me")
}