package audit

import (
	"colossa-pm/helpers"
	"colossa-pm/models"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const auditServiceKey = "auditService"
const auditEntryKey = "auditEntry"

// Middleware injects the audit service into every request context and
// automatically records request metadata after the handler completes.
func Middleware(svc Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		entry := &Entry{
			Request: &models.RequestMeta{
				IPAddress: clientIP(c),
				UserAgent: c.Request.UserAgent(),
				Method:    c.Request.Method,
				URL:       c.Request.RequestURI,
			},
		}

		c.Set(auditServiceKey, svc)
		c.Set(auditEntryKey, entry)

		c.Next()
		setUser(c)
		Flush(c)
	}
}

// Flush writes the current audit entry to the DB.
// Called automatically by the middleware after the handler returns,
// but can also be called manually if needed.
func Flush(c *gin.Context) {
	svc, entry := fromContext(c)
	if svc == nil || entry == nil || entry.Action == "" {
		return
	}
	svc.Log(*entry)
}

// SetAction sets the action name on the current request's audit entry.
//
// Example: audit.SetAction(c, "user.registered")
func SetAction(c *gin.Context, action string) {
	if entry := getEntry(c); entry != nil {
		entry.Action = action
	}
}

func SetAuthUser(c *gin.Context, userID uuid.UUID, fullName string) {
	entry := getEntry(c)
	if entry == nil {
		return
	}
	// Only set if not already populated by setUser from claims
	if entry.UserID == nil {
		entry.UserID = UUIDPtr(userID)
		entry.FullName = fullName
	}
}

// SetUser attaches the authenticated user ID to the current audit entry.
func setUser(c *gin.Context) {
	entry := getEntry(c)
	if entry == nil {
		return
	}
	claims, ok := helpers.GetClaims(c)
	if !ok {
		return
	}
	entry.UserID = UUIDPtr(claims.UserID)
	entry.FullName = claims.FullName
}

// SetEntity attaches the affected entity type and ID to the current audit entry.
//
// Example: audit.SetEntity(c, "user", user.ID)
func SetEntity(c *gin.Context, entityType string, entityID uuid.UUID) {
	if entry := getEntry(c); entry != nil {
		entry.EntityType = StrPtr(entityType)
		entry.EntityID = UUIDPtr(entityID)
	}
}

// SetDiff attaches old and new values to the current audit entry.
//
// Example: audit.SetDiff(c, audit.JSON{"password": "[redacted]"}, audit.JSON{"password": "[redacted]"})
func SetDiff(c *gin.Context, oldValues, newValues models.JSON) {
	if entry := getEntry(c); entry != nil {
		if oldValues != nil {
			entry.OldValues = oldValues
		}
		if newValues != nil {
			entry.NewValues = newValues
		}
	}
}

// --- Internal helpers ---

func getEntry(c *gin.Context) *Entry {
	val, exists := c.Get(auditEntryKey)
	if !exists {
		return nil
	}
	entry, _ := val.(*Entry)
	return entry
}

func fromContext(c *gin.Context) (Service, *Entry) {
	svcVal, exists := c.Get(auditServiceKey)
	if !exists {
		return nil, nil
	}
	svc, _ := svcVal.(Service)
	return svc, getEntry(c)
}

// clientIP extracts the real client IP, respecting X-Forwarded-For
func clientIP(c *gin.Context) string {
	if forwarded := c.GetHeader("X-Forwarded-For"); forwarded != "" {
		parts := strings.SplitN(forwarded, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	return c.RemoteIP()
}
