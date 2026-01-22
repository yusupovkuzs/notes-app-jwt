package main

import (
	"github/yusupovkuzs/GoNotesApp/internal/config"
	"github/yusupovkuzs/GoNotesApp/internal/handlers"
	mwLogger "github/yusupovkuzs/GoNotesApp/internal/middleware"
	"github/yusupovkuzs/GoNotesApp/internal/storage"
	"github/yusupovkuzs/GoNotesApp/internal/storage/postgres"
	"github/yusupovkuzs/GoNotesApp/pkg/logger"
	"github/yusupovkuzs/GoNotesApp/pkg/logger/sl"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// config
	cfg := config.MustLoad()
	// logger
	log := logger.SetupLogger(cfg.Env)
	log.Info("Starting Notes App", slog.String("env", cfg.Env))
	log.Debug("Debug messages are enabled")

	// storage
	database, err := storage.NewStoragePostgres(cfg.Postgres)
	if err != nil {
		log.Error("DB connection failed: %w", sl.Err(err))
		os.Exit(1)
	}
	log.Info("Database connected successfully")

	// migrations
	if err = storage.RunMigrations(database.DB); err != nil {
		log.Error("Migrations failed: %w", sl.Err(err))
		os.Exit(1)
	}
	log.Info("Migrations completed")

	// router
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(mwLogger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	noteRepo := postgres.NewNoteRepoPostgres(database.DB)
	userRepo := postgres.NewUserRepoPostgres(database.DB)
	handler := handlers.NewHandlers(noteRepo, userRepo)

	router.Route("/auth", func(r chi.Router) {
		r.Post("/register", handler.Register(log))
		r.Post("/login", handler.Login(log))
	})

	router.Route("/users", func(r chi.Router) {
		r.Use(handler.UserIdentity(log))
		r.Post("/notes", handler.CreateNote(log))
		r.Get("/notes", handler.GetAllNotes(log))
		r.Get("/notes/{note_id}", handler.GetNote(log))
		r.Put("/notes/{note_id}", handler.UpdateNote(log))
		r.Delete("/notes/{note_id}", handler.DeleteNote(log))
	})

	// start server
	log.Info("starting server", slog.String("address", cfg.HttpServer.Address))
	srv := &http.Server{
		Addr:         cfg.HttpServer.Address,
		Handler:      router,
		ReadTimeout:  cfg.HttpServer.ReadTimeout,
		WriteTimeout: cfg.HttpServer.WriteTimeout,
	}

	if err = srv.ListenAndServe(); err != nil {
		log.Error("failed to stop server")
		return
	}

	log.Info("server stopped")
}
