package server

import (
	"context"
	"fmt"
	"github.com/deepmap/oapi-codegen/pkg/securityprovider"
	restApi "github.com/hazzardr/spacetraders/generated/api"
	"github.com/hazzardr/spacetraders/generated/domain"
	spaceTraders "github.com/hazzardr/spacetraders/generated/spacetraders"
	"github.com/hazzardr/spacetraders/server/handlers"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
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
	DB      *pgxpool.Pool
	Queries *domain.Queries
}

func (dbo *DatabaseOperations) HandlePGError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == pgerrcode.UniqueViolation {
			return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Cannot insert new record. code=%s, message=%s", pgErr.Code, pgErr.Message))
		} else if pgerrcode.IsConnectionException(pgErr.Code) {
			return echo.NewHTTPError(http.StatusServiceUnavailable, fmt.Sprintf("Database connection error, please try again later. code=%s, message=%s", pgErr.Code, pgErr.Message))
		} else if pgerrcode.IsDataException(pgErr.Code) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, fmt.Sprintf("Failure processing request. code=%s, message=%s", pgErr.Code, pgErr.Message))
		} else {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Unhandled Postgres error. code=%s message=%s", pgErr.Code, pgErr.Message))
		}
	}

	return err
}

type Routes struct {
	AgentsHandler *handlers.AgentsHandler
	ShipsHandler  *handlers.ShipsHandler
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

	return &DatabaseOperations{DB: conn, Queries: q}, nil
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
		&handlers.AgentsHandler{
			DBOperations:      dbo,
			SpaceTraderClient: stc,
		},
		&handlers.ShipsHandler{
			DBOperations:      dbo,
			SpaceTraderClient: stc,
		},
	})

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	slog.Info("Starting server on port 8080")
	go func() {
		log.Fatal(e.Start(":8080"))
	}()

	<-stop
	dbo.DB.Close()
	log.Println("Server stopped")
}

func (r Routes) CreateAgent(ctx echo.Context) error {
	return r.AgentsHandler.CreateAgent(ctx)
}

func (r Routes) GetAgentCallSign(ctx echo.Context, callSign string) error {
	return r.AgentsHandler.GetAgentCallSign(ctx, callSign)
}

func (r Routes) GetShipShipId(ctx echo.Context, shipId int) error {
	return r.ShipsHandler.GetShipShipId(ctx, shipId)
}
