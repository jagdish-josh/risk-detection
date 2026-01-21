package auth

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	Email     string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	Password  string    `gorm:"type:varchar(255);not null"`
	Role      string    `gorm:"type:varchar(20);not null;check:role IN ('USER','ADMIN')"`
	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

type UserSecurity struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	DeviceID  string    `gorm:"type:varchar(255);not null"`
	IPAddress string    `gorm:"type:varchar(45);not null"`
	UpdatedAt time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

