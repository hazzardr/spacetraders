package handlers

import (
	"fmt"
	openapi_types "github.com/deepmap/oapi-codegen/pkg/types"
	restApi "github.com/hazzardr/spacetraders/generated/api"
	"github.com/hazzardr/spacetraders/generated/domain"
	spaceTraders "github.com/hazzardr/spacetraders/generated/spacetraders"
	"github.com/hazzardr/spacetraders/server"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

type AgentsHandler struct {
	SpaceTraderClient *spaceTraders.ClientWithResponses
	DBOperations      *server.DatabaseOperations
}

func (h AgentsHandler) CreateAgent(ctx echo.Context) error {
	a := new(restApi.AgentRequest)
	if err := ctx.Bind(a); err != nil {
		return err
	}

	// Should eventually check for bounds on `int` vs `int32` (32 is db type)
	if a.Credits == nil {
		a.Credits = new(int)
		*a.Credits = 0
	}

	if a.ExpiresOn == nil {
		response, err := h.SpaceTraderClient.GetStatusWithResponse(ctx.Request().Context())
		if err != nil {
			return err
		}
		if nil == response {
			return echo.NewHTTPError(http.StatusInternalServerError, "SpaceTraders API returned nil response")
		}
		if response.HTTPResponse.StatusCode != http.StatusOK {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch status from SpaceTraders API responseCode=%d message=%s", response.HTTPResponse.StatusCode, response.HTTPResponse.Status))
		}

		nextReset, err := time.Parse(time.RFC3339, response.JSON200.ServerResets.Next)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, "No expires_on provided and unable to parse next reset time from SpaceTraders API")
		}

		a.ExpiresOn = new(openapi_types.Date)
		*a.ExpiresOn = openapi_types.Date{Time: nextReset}
	}

	date := pgtype.Date{Time: a.ExpiresOn.Time, Valid: true}
	agent, err := h.DBOperations.Queries.InsertAgent(ctx.Request().Context(), domain.InsertAgentParams{
		CallSign:     a.CallSign,
		Faction:      a.Faction,
		Headquarters: a.Headquarters,
		Credits:      int32(*a.Credits),
		ExpiresOn:    date,
	})

	if err != nil {
		return h.DBOperations.HandlePGError(err)
	}

	return ctx.JSON(http.StatusCreated, agent)
}

func (h AgentsHandler) GetAgentCallSign(ctx echo.Context, sign string) error {
	//TODO: clean input
	agent, err := h.DBOperations.Queries.GetAgentByCallsign(ctx.Request().Context(), sign)
	if err != nil {
		return h.DBOperations.HandlePGError(err)
	}

	return ctx.JSON(http.StatusOK, agent)
}
