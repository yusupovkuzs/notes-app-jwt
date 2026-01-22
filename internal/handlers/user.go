package handlers

import (
	"context"
	"errors"
	"github/yusupovkuzs/GoNotesApp/internal/models"
	"github/yusupovkuzs/GoNotesApp/internal/storage"
	"github/yusupovkuzs/GoNotesApp/pkg/logger/sl"
	"github/yusupovkuzs/GoNotesApp/pkg/response"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type Response struct {
	response.Response
	ID int `json:"id,omitempty"`
}

const authorizationHeader = "Authorization"

func (h *Handlers) UserIdentity(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			const op = "handlers.UserIdentity"

			log = log.With(
				slog.String("op", op),
				slog.String("request_id", middleware.GetReqID(r.Context())),
			)

			header := r.Header.Get(authorizationHeader)
			if header == "" {
				log.Error("empty authorization header")
				render.JSON(w, r, response.Error("empty authorization header"))
				response.RespondError(w, http.StatusUnauthorized, "empty authorization header")
				return
			}

			parts := strings.Split(header, " ")
			if len(parts) != 2 {
				log.Error("invalid authorization header")
				render.JSON(w, r, response.Error("invalid authorization header"))
				response.RespondError(w, http.StatusUnauthorized, "invalid authorization header")
				return
			}

			userId, err := h.userRepo.ParseToken(parts[1])
			if err != nil {
				log.Error("invalid token", sl.Err(err))
				render.JSON(w, r, response.Error("invalid token"))
				response.RespondError(w, http.StatusUnauthorized, err.Error())
				return
			}

			log.Info("user identity found", slog.Int("userId", userId))
			ctx := context.WithValue(r.Context(), "userId", userId)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (h *Handlers) Register(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Register"

		var input models.User

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		err := render.DecodeJSON(r.Body, &input)
		if err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			response.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}
		log.Info("request body decoded successfully", slog.Any("request", input))

		if input.Username == "" || input.Password == "" {
			log.Error("invalid username or password")
			response.RespondError(w, http.StatusBadRequest, "invalid username or password")
			return
		}

		id, err := h.userRepo.CreateUser(input)
		if errors.Is(err, storage.ErrUsernameTaken) {
			log.Error("username is already taken", sl.Err(err))
			response.RespondError(w, http.StatusBadRequest, "username is already taken")
			return
		}
		if err != nil {
			log.Error("failed to create user", sl.Err(err))
			response.RespondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		log.Info("user created successfully", slog.Int("id", id))
		response.RespondJSON(w, http.StatusCreated, map[string]interface{}{
			"status": "OK",
			"id":     id,
		})
	}
}

type signInInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *Handlers) Login(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Login"

		var input signInInput

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		err := render.DecodeJSON(r.Body, &input)
		if err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			response.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}
		log.Info("request body decoded successfully", slog.Any("request", input))

		token, err := h.userRepo.GenerateToken(input.Username, input.Password)
		if err != nil {
			log.Error("failed to create token", sl.Err(err))
			response.RespondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		log.Info("token generated", slog.String("token", token))
		response.RespondJSON(w, http.StatusOK, map[string]interface{}{
			"status": "OK",
			"token":  token,
		})
	}
}
