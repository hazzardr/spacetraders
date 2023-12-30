package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/deepmap/oapi-codegen/pkg/securityprovider"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	restApi "github.com/hazzardr/spacetraders/generated/api"
	"github.com/hazzardr/spacetraders/generated/domain"
	spaceTraders "github.com/hazzardr/spacetraders/generated/spacetraders"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
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

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	h := restApi.HandlerFromMux(&RestHandler{operations: dbo}, r)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	slog.Info("Starting server on port 8080")
	go func() {
		log.Fatal(http.ListenAndServe(":8080", h))
	}()

	<-stop
	dbo.db.Close()
	log.Println("Server stopped")
}

type RestHandler struct {
	operations *DatabaseOperations
}

func (rh RestHandler) CreateAgent(w http.ResponseWriter, r *http.Request) {
	var req restApi.AgentRequest
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	err := decoder.Decode(&req)
	if err != nil {
		http.Error(w, "failed to decode request body", http.StatusBadRequest)
		return
	}

	if req.Credits == nil {
		dc := 0
		req.Credits = &dc
	}

	agent, err := rh.operations.queries.InsertAgent(r.Context(), domain.InsertAgentParams{
		CallSign:     req.CallSign,
		Faction:      req.Faction,
		Headquarters: req.Headquarters,
		Credits:      0,
		ExpiresOn:    pgtype.Date{},
	})

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				http.Error(w, "agent already exists", http.StatusConflict)
				return
			}
			if pgErr.Code == pgerrcode.NotNullViolation {
				http.Error(w, "required field is missing", http.StatusBadRequest)
				return
			}
			if pgErr.Code == pgerrcode.CheckViolation {
				http.Error(w, "invalid field value", http.StatusBadRequest)
				return
			}
			http.Error(w, "database error", http.StatusServiceUnavailable)
			return
		} else {
			http.Error(w, fmt.Sprintf("database error %w", err), http.StatusServiceUnavailable)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(agent)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

}

func (rh RestHandler) GetAgentCallSign(w http.ResponseWriter, r *http.Request, callSign string) {
	//TODO implement me
	panic("implement me")
}

func (rh RestHandler) GetShipShipId(w http.ResponseWriter, r *http.Request, shipId int) {
	//TODO implement me
	panic("implement me")
}
