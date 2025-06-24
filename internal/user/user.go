package user

import (
	"errors"
	"github.com/Ryeom/board-game/infra/db"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"

	"github.com/Ryeom/board-game/internal/util" // util 패키지 임포트 추가
)

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type User struct {
	ID           uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	Email        string     `gorm:"uniqueIndex;not null"`
	Password     string     `gorm:"not null"`
	Nickname     string     `gorm:"not null"`
	ProfileImage *string    `gorm:"type:text"`                       // 프로필 사진 URL
	Role         Role       `gorm:"type:varchar(20);default:'user'"` // 권한
	IsActive     bool       `gorm:"default:true"`                    // 탈퇴 여부
	LastLoginAt  *time.Time // 마지막 로그인 시간
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func FindUserByID(id string) (*User, error) {
	var user User
	if err := db.DB.First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 존재하지 않음
		}
		return nil, err
	}
	return &user, nil
}

func FindUserByEmail(email string) (*User, error) {
	var user User
	if err := db.DB.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func CheckPassword(input, hashed string) bool {
	return util.CheckPasswordHash(input, hashed)
}

func SoftDeleteUser(db *gorm.DB, id string) error {
	return db.Model(&User{}).Where("id = ?", id).Update("is_active", false).Error
}

func UpdateLastLoginAt(db *gorm.DB, id string) error {
	return db.Model(&User{}).Where("id = ?", id).Update("last_login_at", time.Now()).Error
}

func IsAdmin(user *User) bool {
	return user.Role == RoleAdmin
}

func HasPermission(user *User, targetID string) bool {
	return user.Role == RoleAdmin || user.ID.String() == targetID
}
