package postgres

import (
	"crypto/sha1"
	"database/sql"
	"errors"
	"fmt"
	"github/yusupovkuzs/GoNotesApp/internal/models"
	"github/yusupovkuzs/GoNotesApp/internal/storage"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/lib/pq"
)

const (
	salt      = "fjewohf7a434gfuoebf9w4"
	signInKey = "oewfuhy6t328yif32g"
	tokenTTL  = time.Hour * 12
)

type UserRepository interface {
	CreateUser(user models.CreateUserRequest) (int, error)
	GetUser(username, password string) (models.User, error)
}

type UserRepoPostgres struct {
	db *sql.DB
}

func NewUserRepoPostgres(db *sql.DB) *UserRepoPostgres {
	return &UserRepoPostgres{db: db}
}

func (r *UserRepoPostgres) CreateUser(u models.User) (int, error) {
	const op = "storage.postgres.CreateUser"

	var id int
	u.Password = generatePasswordHash(u.Password)

	query := fmt.Sprintf(
		"INSERT INTO %s (username, password_hash) VALUES ($1, $2) RETURNING id",
		storage.UsersTable,
	)
	if err := r.db.QueryRow(query, u.Username, u.Password).Scan(&id); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			switch pqErr.Code {
			case "23505":
				return 0, storage.ErrUsernameTaken
			}
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

func (r *UserRepoPostgres) GetUser(username, password string) (models.User, error) {
	const op = "storage.postgres.GetUser"

	var user models.User

	query := fmt.Sprintf(
		"SELECT id, created_at FROM %s WHERE username = $1 AND password_hash = $2",
		storage.UsersTable,
	)
	if err := r.db.QueryRow(query, username, password).Scan(&user.ID, &user.CreatedAt); err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	user.Username = username
	user.Password = password
	return user, nil
}

func generatePasswordHash(password string) string {
	hash := sha1.New()
	hash.Write([]byte(password))

	return fmt.Sprintf("%x", hash.Sum([]byte(salt)))
}

type tokenClaims struct {
	jwt.StandardClaims
	UserId int `json:"user_id"`
}

func (r *UserRepoPostgres) GenerateToken(username, password string) (string, error) {
	user, err := r.GetUser(username, generatePasswordHash(password))
	if err != nil {
		return "", err
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims{
		jwt.StandardClaims{
			ExpiresAt: time.Now().Add(tokenTTL).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
		user.ID,
	})

	return token.SignedString([]byte(signInKey))
}

func (r *UserRepoPostgres) ParseToken(accessToken string) (int, error) {
	token, err := jwt.ParseWithClaims(accessToken, &tokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}

		return []byte(signInKey), nil
	})
	if err != nil {
		return 0, err
	}

	claims, ok := token.Claims.(*tokenClaims)
	if !ok {
		return 0, errors.New("invalid token claims")
	}

	return claims.UserId, nil
}
