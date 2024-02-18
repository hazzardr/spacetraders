package handlers

import (
	"context"
	"fmt"
	openapi_types "github.com/deepmap/oapi-codegen/pkg/types"
	restApi "github.com/hazzardr/spacetraders/generated/api"
	"github.com/hazzardr/spacetraders/generated/domain"
	spaceTraders "github.com/hazzardr/spacetraders/generated/spacetraders"
	"github.com/hazzardr/spacetraders/server/database"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

// AgentsAPIHandler handles requests for the /agents endpoints. Its package public functions serve traditional REST endpoints served with JSON.
// These should adhere to the OpenAPI spec via the generated code satisfying the interface defined in server/generated/api/server_gen.go.
type AgentsAPIHandler struct {
	SpaceTraderClient *spaceTraders.ClientWithResponses
	DBOperations      *database.Operations
}

// CreateAgent creates a new agent and returns the agent in JSON format
func (h AgentsAPIHandler) CreateAgent(ctx echo.Context) error {
	a := new(restApi.AgentRequest)
	if err := ctx.Bind(a); err != nil {
		return err
	}

	agent, err := h.createAgent(ctx.Request().Context(), *a)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusCreated, agent)
}

func (h AgentsAPIHandler) createAgent(ctx context.Context, a restApi.AgentRequest) (domain.SpacetradersAgent, error) {
	// Should eventually check for bounds on `int` vs `int32` (32 is db type)
	if a.Credits == nil {
		a.Credits = new(int)
		*a.Credits = 0
	}

	if a.ExpiresOn == nil {
		response, err := h.SpaceTraderClient.GetStatusWithResponse(ctx)
		if err != nil {
			return domain.SpacetradersAgent{}, err
		}
		if nil == response {
			return domain.SpacetradersAgent{}, echo.NewHTTPError(http.StatusInternalServerError, "SpaceTraders API returned nil response")
		}
		if response.HTTPResponse.StatusCode != http.StatusOK {
			return domain.SpacetradersAgent{}, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch status from SpaceTraders API responseCode=%d message=%s", response.HTTPResponse.StatusCode, response.HTTPResponse.Status))
		}

		nextReset, err := time.Parse(time.RFC3339, response.JSON200.ServerResets.Next)
		if err != nil {
			return domain.SpacetradersAgent{}, echo.NewHTTPError(http.StatusUnprocessableEntity, "No expires_on provided and unable to parse next reset time from SpaceTraders API")
		}

		a.ExpiresOn = new(openapi_types.Date)
		*a.ExpiresOn = openapi_types.Date{Time: nextReset}
	}

	date := pgtype.Date{Time: a.ExpiresOn.Time, Valid: true}
	agent, err := h.DBOperations.Queries.InsertAgent(ctx, domain.InsertAgentParams{
		CallSign:     a.CallSign,
		Faction:      a.Faction,
		Headquarters: a.Headquarters,
		Credits:      int32(*a.Credits),
		ExpiresOn:    date,
	})

	if err != nil {
		return domain.SpacetradersAgent{}, h.DBOperations.HandlePGError(err)
	}

	return agent, nil
}

// GetAgentCallSign returns the agent with the given call sign in JSON format
func (h AgentsAPIHandler) GetAgentCallSign(ctx echo.Context, sign string) error {
	//TODO: clean input
	agent, err := h.getAgentByCallSign(ctx.Request().Context(), sign)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, agent)
}

// GetAgentByCallSign returns the agent with the given call sign
func (h AgentsAPIHandler) getAgentByCallSign(ctx context.Context, sign string) (domain.SpacetradersAgent, error) {
	agent, err := h.DBOperations.Queries.GetAgentByCallsign(ctx, sign)
	if err != nil {
		return domain.SpacetradersAgent{}, h.DBOperations.HandlePGError(err)
	}

	return agent, nil
}

// ListAgents returns all agents in JSON format
func (h AgentsAPIHandler) ListAgents(ctx echo.Context) error {
	agents, err := h.listAgents(ctx.Request().Context())
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, agents)
}

// ListAgents returns all agents
func (h AgentsAPIHandler) listAgents(ctx context.Context) ([]domain.SpacetradersAgent, error) {
	agents, err := h.DBOperations.Queries.ListAgents(ctx)
	if err != nil {
		return nil, h.DBOperations.HandlePGError(err)
	}

	return agents, nil
}

// RefreshAgentCallSign refreshes the agent with the given call sign and returns the agent in JSON format.
func (h AgentsAPIHandler) RefreshAgentCallSign(ctx echo.Context, sign string) error {
	agent, err := h.refreshAgentByCallSign(ctx.Request().Context(), sign)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, agent)
}

// refreshAgentByCallSign refreshes the agent with the given call sign by calling the SpaceTraders API to get the latest agent info.
// If the current agent is expired, it creates a new one
func (h AgentsAPIHandler) refreshAgentByCallSign(ctx context.Context, sign string) (domain.SpacetradersAgent, error) {
	resp, err := h.SpaceTraderClient.GetAgentWithResponse(ctx, sign)

	if err != nil {
		return domain.SpacetradersAgent{}, err
	}
	if nil == resp {
		return domain.SpacetradersAgent{}, echo.NewHTTPError(http.StatusInternalServerError, "SpaceTraders API returned nil response")
	}
	if resp.HTTPResponse.StatusCode == http.StatusNotFound {
		// Agent doesn't exist, create a new one with spacetraders
		email := "rbrianhazzard+spacetraders@gmail.com"
		req := spaceTraders.RegisterJSONRequestBody{
			Email:   &email,
			Faction: "COSMIC",
			Symbol:  sign,
		}
		response, err := h.SpaceTraderClient.RegisterWithResponse(ctx, req)
		if err != nil {
			return domain.SpacetradersAgent{}, err
		}
		if nil == response {
			return domain.SpacetradersAgent{}, echo.NewHTTPError(http.StatusInternalServerError, "SpaceTraders API returned nil response when creating agent")
		}
		if response.HTTPResponse.StatusCode != http.StatusCreated {
			return domain.SpacetradersAgent{}, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create agent with SpaceTraders API responseCode=%d message=%s", response.HTTPResponse.StatusCode, response.HTTPResponse.Status))
		}

		rd := response.JSON201.Data
		credits := new(int)
		*credits = int(rd.Agent.Credits)

		agent, err := h.createAgent(ctx, restApi.AgentRequest{
			CallSign:     sign,
			Faction:      rd.Faction.Name,
			Headquarters: rd.Agent.Headquarters,
			Credits:      credits,
		})
		if err != nil {
			return domain.SpacetradersAgent{}, err
		}
		return agent, nil

	}
	if resp.HTTPResponse.StatusCode != http.StatusOK {
		return domain.SpacetradersAgent{}, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch status from SpaceTraders API responseCode=%d message=%s", resp.HTTPResponse.StatusCode, resp.HTTPResponse.Status))
	}

	agent, err := h.getAgentByCallSign(ctx, resp.JSON200.Data.Symbol)

	return agent, nil
}
