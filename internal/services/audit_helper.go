package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func writeAudit(
	ctx context.Context,
	tx pgx.Tx,
	actorID uuid.UUID,
	action string,
	entityType string,
	entityID string,
	oldValues *string,
	newValues *string,
	ip *string,
) error {

	_, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (
			id,
			actor_id,
			action,
			entity_type,
			entity_id,
			old_values,
			new_values,
			ip_address,
			created_at
		)
		VALUES (
			uuid_generate_v4(),
			$1,$2,$3,$4,$5,$6,$7,NOW()
		)
	`,
		actorID,
		action,
		entityType,
		entityID,
		oldValues,
		newValues,
		ip,
	)

	return err
}
