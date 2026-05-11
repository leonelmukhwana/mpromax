package models

import (
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID         uuid.UUID `db:"id"`
	ActorID    uuid.UUID `db:"actor_id"`
	Action     string    `db:"action"`
	EntityType string    `db:"entity_type"`
	EntityID   string    `db:"entity_id"`

	OldValues *string `db:"old_values"`
	NewValues *string `db:"new_values"`

	IPAddress *string   `db:"ip_address"`
	CreatedAt time.Time `db:"created_at"`
}
