package middleware

import (
	"errors"
	"net/http"
)

const (
	userCtx = "userId"
)

func GetUserID(r *http.Request) (int, error) {
	id := r.Context().Value(userCtx)
	if id == nil {
		return 0, errors.New("user is not found")
	}

	idInt, ok := id.(int)
	if !ok {
		return 0, errors.New("user id is of invalid type")
	}

	return idInt, nil
}
