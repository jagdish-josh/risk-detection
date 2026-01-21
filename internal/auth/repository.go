package auth

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) FindUserByID(userID uuid.UUID) (*User, error) {
	
	var user User

	err := r.db.Where("id = ?", userID).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &user, err
}

func (r *repository) UpdateUserSecurity( userID uuid.UUID, deviceID string, ipAdress string) error {
	var userSec UserSecurity
	err := r.db.Where("user_id = ?", userID).First(&userSec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Entry does not exist, create it
		userSec = UserSecurity{
			UserID:    userID,
			DeviceID:  deviceID,
			IPAddress: ipAdress,
			UpdatedAt: time.Now(),
		}
		return r.db.Create(&userSec).Error
	} else if err != nil {
		return err
	}

	// Entry exists, update it
	return r.db.Model(&UserSecurity{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"device_id":  deviceID,
			"ip_address": ipAdress,
			"updated_at": time.Now(),
		}).Error

}


