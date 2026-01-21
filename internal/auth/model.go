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

type LoginRequest struct {
	UserID   uuid.UUID `json:"user_id" binding:"required,uuid"`
	Password string    `json:"password" binding:"required"`
	DeviceID string    `json:"device_id" binding:"required"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
}

type Repository interface {
	FindUserByID(userID uuid.UUID) (*User, error)
	UpdateUserSecurity(uuid.UUID, string, string) error
}

type Service interface {
	Login(req LoginRequest, ipAddress string) (LoginResponse, error)
}
func (UserSecurity) TableName() string {
    return "user_security"
}
