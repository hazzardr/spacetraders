package server

import (
	"context"
	"fmt"
	"github.com/deepmap/oapi-codegen/pkg/securityprovider"
	openapi_types "github.com/deepmap/oapi-codegen/pkg/types"
	restApi "github.com/hazzardr/spacetraders/generated/api"
	"github.com/hazzardr/spacetraders/generated/domain"
	spaceTraders "github.com/hazzardr/spacetraders/generated/spacetraders"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"
)

type Config struct {
	DatabaseUrl         string `mapstructure:"DATABASE_URL"`
	SpaceTradersBaseUrl string `mapstructure:"SPACE_TRADERS_BASE_URL"`
	AgentToken          string `mapstructure:"AGENT_TOKEN"`
}

type DatabaseOperations struct {
	db      *pgxpool.Pool
	queries *domain.Queries
}

func newDBO(dbUrl string) (*DatabaseOperations, error) {
	conn, err := pgxpool.New(context.Background(), dbUrl)
	if err != nil {
		return nil, err
	}

	err = conn.Ping(context.Background())
	if err != nil {
		return nil, err
	}

	q := domain.New(conn)

	return &DatabaseOperations{db: conn, queries: q}, nil
}

func newSpaceTradersClient(config Config) (*spaceTraders.ClientWithResponses, error) {
	client := http.Client{
		Timeout: 30 * time.Second,
	}
	tokenProvider, err := securityprovider.NewSecurityProviderBearerToken(config.AgentToken)
	if err != nil {
		return nil, err
	}

	return spaceTraders.NewClientWithResponses(
		config.SpaceTradersBaseUrl,
		spaceTraders.WithRequestEditorFn(tokenProvider.Intercept),
		spaceTraders.WithHTTPClient(&client))
}

func StartServer() {
	viper.SetConfigFile(".env")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("failed to load configuration, error=%w", err))
	}

	var config Config
	err = viper.Unmarshal(&config)
	if err != nil {
		panic(fmt.Errorf("failed to load configuration, error=%w", err))
	}

	e := echo.New()
	e.Logger.SetHeader("${time_rfc3339} ${level}")
	e.Use(middleware.Logger(), middleware.Recover(), middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		ErrorMessage: "Request timed out",
		Timeout:      60 * time.Second,
	}))

	dbo, err := newDBO(config.DatabaseUrl)
	if err != nil {
		panic(err)
	}

	stc, err := newSpaceTradersClient(config)

	restApi.RegisterHandlers(e, &Routes{
		DBOperations:      dbo,
		SpaceTraderClient: stc,
	})

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	slog.Info("Starting server on port 8080")
	go func() {
		log.Fatal(e.Start(":8080"))
	}()

	<-stop
	dbo.db.Close()
	log.Println("Server stopped")
}

type Routes struct {
	SpaceTraderClient *spaceTraders.ClientWithResponses
	DBOperations      *DatabaseOperations
}

func (r Routes) CreateAgent(ctx echo.Context) error {
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
		response, err := r.SpaceTraderClient.GetStatusWithResponse(ctx.Request().Context())
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
	agent, err := r.DBOperations.queries.InsertAgent(ctx.Request().Context(), domain.InsertAgentParams{
		CallSign:     a.CallSign,
		Faction:      a.Faction,
		Headquarters: a.Headquarters,
		Credits:      int32(*a.Credits),
		ExpiresOn:    date,
	})

	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusCreated, agent)
}

func (r Routes) GetAgentCallSign(ctx echo.Context, callSign string) error {
	//TODO: clean input
	agent, err := r.DBOperations.queries.GetAgentByCallsign(ctx.Request().Context(), callSign)
	if err != nil {
		return handlePGError(err)
	}

	return ctx.JSON(http.StatusOK, agent)
}

func (r Routes) GetShipShipId(ctx echo.Context, shipId int) error {
	//TODO implement me
	panic("implement me")
}

func handlePGError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == pgerrcode.UniqueViolation {
			return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Cannot insert new record: %s", pgErr.Message))
		} else if pgerrcode.IsConnectionException(pgErr.Code) {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "Database connection error, please try again later")
		} else if pgerrcode.IsDataException(pgErr.Code) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, "Failure processing request")
		} else {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Unhandled Postgres error code=%s message=%s", pgErr.Code, pgErr.Message))
		}
	}

	return err
}
