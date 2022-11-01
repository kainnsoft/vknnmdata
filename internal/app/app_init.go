package app

import (
	"database/sql"
	"fmt"
	config "mdata/configs"
	"mdata/internal/repository"
	"mdata/internal/routes"
	"mdata/pkg/httpserver"
	"mdata/pkg/logging"
	"mdata/pkg/pg"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/pressly/goose/v3"
)

func Run(cfg *config.Config) {
	cfgLog := cfg.Log
	log := logging.NewMDLogger(cfgLog.InfoPath, cfgLog.ErrorPath, cfgLog.BuhPath)

	// db
	insPgDB := includePg(cfg, log)
	if insPgDB != nil {
		log.Infof("app - Run - db - Ok!")
		defer insPgDB.Close()
	}

	// HTTP Server
	mux := http.NewServeMux()

	ins := &repository.PostgreInstance{Db: insPgDB.Pool}

	routes.InitializeRoutes(mux, ins)

	// addr := flag.String("addr", ":8080", "Сетевой адрес веб-сервера MD")
	// flag.Parse()
	// err := http.ListenAndServe(*addr, mux)
	// log.Errorf("main: srv.ListenAndServe() error: %v", err)

	// v1.NewRouter(mux, inUseCase, grpcClient, log)

	httpServer := httpserver.New(mux, cfg.HTTP)
	if httpServer != nil {
		log.Infof("app - Run - httpServer has run on addr %v", httpServer.GetAddr())
	}

	//------------------------------------

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case s := <-interrupt:
		log.Infof("app - Run - signal: " + s.String())
	case err := <-httpServer.Notify():
		log.Errorf("app - Run - httpServer.Notify: %w", err)
	}

	// Shutdown
	err := httpServer.Shutdown()
	if err != nil {
		log.Errorf("app - Run - httpServer.Shutdown: %w", err)
	}

}

func includePg(cfg *config.Config, log *logging.MDLogger) *pg.DB {
	strurl := fmt.Sprintf("%s://%s:%s@%s:%d/%s?sslmode=disable&connect_timeout=%d",
		"postgres",
		url.QueryEscape(cfg.PG.Username),
		url.QueryEscape(cfg.PG.Password),
		cfg.PG.Host,
		cfg.PG.Port,
		cfg.PG.DBName,
		cfg.PG.ConnTimeout)

	insPgDB, err := pg.NewInsPgDB(strurl, cfg.PG.PoolMax)
	if err != nil {
		log.Fatalf("Can't create DB connection: %v", err)

		return nil
	}

	// migrationUp(strurl, log)

	return insPgDB
}

func migrationUp(strurl string, log *logging.MDLogger) {
	conn, err := sql.Open("postgres", strurl)
	if err != nil {
		log.Fatalf("Can't sql.Open migrarion: %v", err)
	}
	err = goose.Up(conn, "migrations")
	if err != nil {
		log.Fatalf("Can't create migrarion: %v", err)
	}
}
