package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/nicholasjackson/env"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel"

	"github.com/gorilla/mux"
	"github.com/hashicorp-demoapp/product-api-go/config"
	"github.com/hashicorp-demoapp/product-api-go/data"
	"github.com/hashicorp-demoapp/product-api-go/handlers"
	"github.com/hashicorp-demoapp/product-api-go/telemetry"
	"github.com/hashicorp/go-hclog"
)

// Config format for application
type Config struct {
	DBConnection   string `json:"db_connection"`
	BindAddress    string `json:"bind_address"`
	MetricsAddress string `json:"metrics_address"`
}

var conf *Config
var logger hclog.Logger

var configFile = env.String("CONFIG_FILE", false, "./conf.json", "Path to JSON encoded config file")

const jwtSecret = "test"

func main() {
	logger = hclog.New(
		&hclog.LoggerOptions{
			Name:       telemetry.SERVICE_NAME,
			JSONFormat: true,
		},
	)
	hclog.SetDefault(logger)

	ctx, closer, err := telemetry.InitTracer()
	if err != nil {
		logger.Error("error initilizing tracer", "error", err)
	}
	defer closer()
	ctx, span := otel.GetTracerProvider().Tracer("product-api-go").Start(ctx, "init")

	err = env.Parse()
	if err != nil {
		logger.Error("Error parsing flags", "error", err)
		os.Exit(1)
	}

	conf = &Config{}

	// load the config
	c, err := config.New(*configFile, conf, configUpdated)
	if err != nil {
		logger.Error("Unable to load config file", "error", err)
		os.Exit(1)
	}
	defer c.Close()

	// configure the telemetry
	//t := telemetry.New(conf.MetricsAddress)

	// load the db connection
	db, err := retryDBUntilReady(ctx)
	if err != nil {
		logger.Error("Timeout waiting for database connection")
		os.Exit(1)
	}

	r := mux.NewRouter()
	r.Use(otelmux.Middleware(telemetry.SERVICE_NAME))

	authMiddleware := handlers.NewAuthMiddleware(db, logger)

	healthHandler := handlers.NewHealth(logger, db)
	r.Handle("/health", healthHandler).Methods("GET")

	coffeeHandler := handlers.NewCoffee(db, logger)

	r.Handle("/coffees", coffeeHandler).Methods("GET")
	r.Handle("/coffees", authMiddleware.IsAuthorized(coffeeHandler.CreateCoffee)).Methods("POST")

	ingredientsHandler := handlers.NewIngredients(db, logger)
	r.Handle("/coffees/{id:[0-9]+}/ingredients", ingredientsHandler).Methods("GET")
	r.Handle("/coffees/{id:[0-9]+}/ingredients", authMiddleware.IsAuthorized(ingredientsHandler.CreateCoffeeIngredient)).Methods("POST")

	userHandler := handlers.NewUser(db, logger)
	r.HandleFunc("/signup", userHandler.SignUp).Methods("POST")
	r.HandleFunc("/signin", userHandler.SignIn).Methods("POST")
	r.HandleFunc("/signout", userHandler.SignOut).Methods("POST")

	orderHandler := handlers.NewOrder(db, logger)
	r.Handle("/orders", authMiddleware.IsAuthorized(orderHandler.GetUserOrders)).Methods("GET")
	r.Handle("/orders", authMiddleware.IsAuthorized(orderHandler.CreateOrder)).Methods("POST")
	r.Handle("/orders/{id:[0-9]+}", authMiddleware.IsAuthorized(orderHandler.GetUserOrder)).Methods("GET")
	r.Handle("/orders/{id:[0-9]+}", authMiddleware.IsAuthorized(orderHandler.UpdateOrder)).Methods("PUT")
	r.Handle("/orders/{id:[0-9]+}", authMiddleware.IsAuthorized(orderHandler.DeleteOrder)).Methods("DELETE")

	logger.Info("Starting service", "bind", conf.BindAddress, "metrics", conf.MetricsAddress)
	span.End()
	err = http.ListenAndServe(conf.BindAddress, r)
	if err != nil {
		logger.Error("Unable to start server", "bind", conf.BindAddress, "error", err)
	}
}

// retryDBUntilReady keeps retrying the database connection
// when running the application on a scheduler it is possible that the app will come up before
// the database, this can cause the app to go into a CrashLoopBackoff cycle
func retryDBUntilReady(ctx context.Context) (data.Connection, error) {
	st := time.Now()
	dt := 1 * time.Second  // this should be an exponential backoff
	mt := 60 * time.Second // max time to wait of the DB connection

	for {
		db, err := data.New(ctx, conf.DBConnection)
		if err == nil {
			return db, nil
		}

		logger.Error("Unable to connect to database", "error", err)

		// check if max time has elapsed
		if time.Now().Sub(st) > mt {
			return nil, err
		}

		// retry
		time.Sleep(dt)
	}
}

func configUpdated() {
	logger.Info("Config file changed")
}
