package helpers

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetUserID extracts the authenticated user's ID from the Gin context.
// Returns 0 and false if not set (i.e. route is not behind Authenticate middleware).
func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	val, exists := c.Get("userID")
	if !exists {
		return uuid.Nil, false
	}
	id, ok := val.(uuid.UUID)
	return id, ok
}
