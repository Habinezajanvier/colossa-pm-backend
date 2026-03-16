package audit

import (
	applogger "colossa-pm/logger"
	"colossa-pm/models"

	"github.com/google/uuid"
)

// Entry is the input used to build an AuditLog — all fields optional except Action
type Entry struct {
	UserID     *uuid.UUID
	FullName   string
	Action     string
	EntityType *string
	EntityID   *uuid.UUID
	OldValues  models.JSON
	NewValues  models.JSON
	Request    *models.RequestMeta
}

// Service writes audit log entries asynchronously
type Service interface {
	Log(entry Entry)
}

type service struct {
	repo AuditRepository
}

func NewService(repo AuditRepository) Service {
	return &service{repo: repo}
}

// Log fires the audit log write in a goroutine — never blocks the caller
func (s *service) Log(entry Entry) {
	go func() {
		log := &models.AuditLogModel{
			UserID:       entry.UserID,
			UserFullName: entry.FullName,
			Action:       entry.Action,
			EntityType:   entry.EntityType,
			EntityID:     entry.EntityID,
			Request:      entry.Request,
		}

		if entry.OldValues != nil {
			log.OldValues = &entry.OldValues
		}
		if entry.NewValues != nil {
			log.NewValues = &entry.NewValues
		}

		if err := s.repo.Save(log); err != nil {
			applogger.Instance().ErrorMsg("failed to save audit log: " + err.Error())
		}
	}()
}

// --- Helpers to build entries cleanly ---

func StrPtr(s string) *string {
	v := s
	return &v
}

func UUIDPtr(u uuid.UUID) *uuid.UUID {
	v := u
	return &v
}
