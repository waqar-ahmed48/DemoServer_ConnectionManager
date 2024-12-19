package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"DemoServer_ConnectionManager/configuration"
	"DemoServer_ConnectionManager/datalayer"
	"DemoServer_ConnectionManager/handlers"
	"DemoServer_ConnectionManager/helper"

	"github.com/go-openapi/runtime/middleware"
	"github.com/gorilla/mux"
	"github.com/ilyakaznacheev/cleanenv"
)

func main() {
	var cfg configuration.Config

	configPath := configuration.ProcessArgs(&cfg)

	// read configuration from the file and environment variables
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}

	/*if _, err := os.Stat(cfg.Configuration.Log_Folder); err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(cfg.Configuration.Log_Folder, 0700)

			if err != nil {
				fmt.Println(err)
				os.Exit(2)
			}
		} else {
			fmt.Println(err)
			os.Exit(2)
		}
	}

	file, err := os.OpenFile(cfg.Configuration.Log_Folder+"/"+cfg.Configuration.Log_File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)

	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	defer file.Close()*/

	w := io.MultiWriter(os.Stdout)

	loggerOpts := &slog.HandlerOptions{
		AddSource: false,
		Level:     slog.LevelDebug,
		/*ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				s := a.Value.Any().(*slog.Source)
				s.File = path.Base(s.File)
			}
			return a
		},*/
	}

	sl := slog.New(slog.NewJSONHandler(w, loggerOpts))

	logAttrGroup := slog.Group(
		"common",
		"service_name", cfg.Server.PrefixMain)

	l := sl.With(logAttrGroup)
	slog.SetDefault(l)

	r := mux.NewRouter()

	pd, err := datalayer.NewPostgresDataSource(&cfg, l)
	if err != nil {
		l.Error("PostgresDataSource initialization failed. Error: " + err.Error())
		os.Exit(2)
	}

	err = pd.AutoMigrate()
	if err != nil {
		l.Error("PostgresDataSource AutoMigration failed. Error: " + err.Error())
		os.Exit(2)
	}

	ch, err := handlers.NewConnectionsHandler(&cfg, l, pd)
	if err != nil {
		l.Error("Connections Handler initialization failed. Error: " + err.Error())
		os.Exit(2)
	}

	sh := handlers.NewStatusHandler(l, pd)

	statusRouter := r.Methods(http.MethodGet).Subrouter()
	statusRouter.HandleFunc("/v1/connectionmgmt/status", sh.GetStatus)

	cGetRouter := r.Methods(http.MethodGet).Subrouter()
	cGetRouter.HandleFunc("/v1/connectionmgmt/connections", ch.GetConnections)
	cGetRouter.Use(ch.MiddlewareValidateConnectionsGet)

	jch, err := handlers.NewAWSConnectionHandler(&cfg, l, pd)
	if err != nil {
		l.Error("AWSConnectionHandler initialization failed. Error: " + err.Error())
		os.Exit(2)
	}

	jcGetConnectionsRouter := r.Methods(http.MethodGet).Subrouter()
	jcGetConnectionsRouter.HandleFunc("/v1/connectionmgmt/connections/aws", jch.GetAWSConnections)
	jcGetConnectionsRouter.Use(jch.MiddlewareValidateAWSConnectionsGet)

	jcGetRouterWithID := r.Methods(http.MethodGet).Subrouter()
	jcGetRouterWithID.HandleFunc("/v1/connectionmgmt/connection/aws/{connectionid:[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$}", jch.GetAWSConnection)
	jcGetRouterWithID.HandleFunc("/v1/connectionmgmt/connection/aws/test/{connectionid:[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$}", jch.TestAWSConnection)
	jcGetRouterWithID.Use(jch.MiddlewareValidateAWSConnection)

	jcPostRouter := r.Methods(http.MethodPost).Subrouter()
	jcPostRouter.HandleFunc("/v1/connectionmgmt/connection/aws", jch.AddAWSConnection)
	jcPostRouter.Use(jch.MiddlewareValidateAWSConnectionPost)

	jcPatchRouter := r.Methods(http.MethodPatch).Subrouter()
	jcPatchRouter.HandleFunc("/v1/connectionmgmt/connection/aws/{connectionid:[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$}", jch.UpdateAWSConnection)
	jcPatchRouter.Use(jch.MiddlewareValidateAWSConnectionUpdate)

	jcDeleteRouter := r.Methods(http.MethodDelete).Subrouter()
	jcDeleteRouter.HandleFunc("/v1/connectionmgmt/connection/aws/{connectionid:[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$}", jch.DeleteAWSConnection)
	jcDeleteRouter.Use(jch.MiddlewareValidateAWSConnection)

	opts := middleware.RedocOpts{SpecURL: "/swagger.yaml"}
	docs_sh := middleware.Redoc(opts, nil)

	docsRouter := r.Methods(http.MethodGet).Subrouter()
	docsRouter.Handle("/docs", docs_sh)
	docsRouter.Handle("/swagger.yaml", http.FileServer(http.Dir("./")))

	s := http.Server{
		Addr:         ":" + strconv.Itoa(cfg.Server.Port),
		Handler:      r,
		IdleTimeout:  time.Duration(cfg.Server.HTTPIdleTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.HTTPWriteTimeout) * time.Second,
		ReadTimeout:  time.Duration(cfg.Server.HTTPReadTimeout) * time.Second,
	}

	go func() {
		l.Info("Started listening", slog.Int("port", cfg.Server.Port))

		err := s.ListenAndServe()
		if err != nil {
			l.Info(err.Error())
			// os.Exit(0)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	signal.Notify(sigChan, syscall.SIGTERM)

	sig := <-sigChan
	l.Info("Terminal request received. Initiating Graceful shutdown", sig)

	tc, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Server.HTTPShutdownTimeout)*time.Second)
	defer cancel()
	l.Info("New requests processing stopped.")

	err = s.Shutdown(tc)
	if err != nil {
		helper.LogError(l, helper.ErrorHTTPServerShutdownFailed, err)
	}
}
