package utils

import (
	"net/http"

	"github.com/conflict-driven-devs/headquarter/internal/models"
)

// GetUserID, GetUserRole, for now unused but could be useful later
func GetUserID(r *http.Request) (string, bool) {
	userID, ok := r.Context().Value(models.UserIDKey).(string)
	return userID, ok
}

func GetUserRole(r *http.Request) (models.UserRole, bool) {
	role, ok := r.Context().Value(models.UserRoleKey).(models.UserRole)
	return role, ok
}
