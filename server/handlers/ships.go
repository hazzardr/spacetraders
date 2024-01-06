package handlers

import (
	spaceTraders "github.com/hazzardr/spacetraders/generated/spacetraders"
	"github.com/hazzardr/spacetraders/server/database"
	"github.com/labstack/echo/v4"
)

type ShipsAPIHandler struct {
	SpaceTraderClient *spaceTraders.ClientWithResponses
	DBOperations      *database.Operations
}

func (h ShipsAPIHandler) GetShipShipId(ctx echo.Context, id int) error {
	//TODO: Implement
	panic("implement me")
}
