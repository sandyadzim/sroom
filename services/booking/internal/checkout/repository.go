package checkout

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) HasAvailableUnit(ctx context.Context, unitIDs []string, pickup, returnAt time.Time) (bool, error) {
	if len(unitIDs) == 0 {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	var blocked int
	err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(DISTINCT unit_id)
		FROM booking.allocations
		WHERE unit_id IN ?
		  AND blocks_inventory
		  AND occupied_period && tstzrange(?, ?, '[)')
	`, unitIDs, pickup, returnAt).Scan(&blocked).Error
	return blocked < len(unitIDs), err
}
