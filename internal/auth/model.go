package auth

import (
	"github.com/google/uuid"
	"time"
)

type User struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	Email     string    `gorm:"uniqueIndex;not null"`
	Password  string    `gorm:"not null"`
	Nickname  string    `gorm:"not null"`
	CreatedAt time.Time
}
