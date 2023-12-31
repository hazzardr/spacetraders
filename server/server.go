package server

import (
	"context"
	"fmt"
	"github.com/deepmap/oapi-codegen/pkg/securityprovider"
	restApi "github.com/hazzardr/spacetraders/generated/api"
	"github.com/hazzardr/spacetraders/generated/domain"
	spaceTraders "github.com/hazzardr/spacetraders/generated/spacetraders"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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

	dbo, err := newDBO(config.DatabaseUrl)
	if err != nil {
		panic(err)
	}

	e := echo.New()
	e.Use(middleware.Logger(), middleware.Recover(), middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		ErrorMessage: "Request timed out",
		Timeout:      60 * time.Second,
	}))

	restApi.RegisterHandlers(e, &Routes{operations: dbo})

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
	operations *DatabaseOperations
}

func (r Routes) CreateAgent(ctx echo.Context) error {
	//TODO implement me
	panic("implement me")
}

func (r Routes) GetAgentCallSign(ctx echo.Context, callSign string) error {
	//TODO implement me
	panic("implement me")
}

func (r Routes) GetShipShipId(ctx echo.Context, shipId int) error {
	//TODO implement me
	panic("implement me")
}
