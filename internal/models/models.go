package models

import (
	"time"

	"gorm.io/gorm"
)

type UserType int

const (
	UserTypeUnverified UserType = iota
	UserTypeUser
	UserTypeModerator
	UserTypeAdmin
)

func (ut UserType) String() string {
	switch ut {
	case UserTypeUnverified:
		return "unverified"
	case UserTypeUser:
		return "user"
	case UserTypeModerator:
		return "moderator"
	case UserTypeAdmin:
		return "admin"
	default:
		return "unknown"
	}
}

type User struct {
	ID           uint     `gorm:"primaryKey"`
	Username     string   `gorm:"unique;not null"`
	Email        string   `gorm:"unique;not null"`
	PasswordHash string   `gorm:"not null"`
	UserType     UserType `gorm:"default:0"`

	// Profile fields
	Motto         string `gorm:"size:255"`
	ProfilePicURL string `gorm:"size:500"`
	Signature     string `gorm:"size:1000"`

	// Email verification
	EmailVerified     bool   `gorm:"default:false"`
	VerificationToken string `gorm:"size:64"`

	// Ban management
	IsBanned    bool `gorm:"default:false"`
	BannedAt    *time.Time
	BannedUntil *time.Time
	BanReason   string `gorm:"size:500"`

	// Timestamps
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Relations
	Posts  []Post  `gorm:"foreignKey:AuthorID"`
	Topics []Topic `gorm:"foreignKey:AuthorID"`
}

type Section struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"not null"`
	Description string `gorm:"size:500"`
	Order       int    `gorm:"default:0"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Relations
	Categories []Category `gorm:"foreignKey:SectionID"`
}

type Category struct {
	ID          uint   `gorm:"primaryKey"`
	SectionID   uint   `gorm:"not null"`
	Name        string `gorm:"not null"`
	Description string `gorm:"size:500"`
	Order       int    `gorm:"default:0"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Relations
	Section Section `gorm:"foreignKey:SectionID"`
	Topics  []Topic `gorm:"foreignKey:CategoryID"`
}

type Topic struct {
	ID         uint   `gorm:"primaryKey"`
	CategoryID uint   `gorm:"not null"`
	AuthorID   uint   `gorm:"not null"`
	Title      string `gorm:"not null"`
	IsPinned   bool   `gorm:"default:false"`
	IsLocked   bool   `gorm:"default:false"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Relations
	Category Category `gorm:"foreignKey:CategoryID"`
	Author   User     `gorm:"foreignKey:AuthorID"`
	Posts    []Post   `gorm:"foreignKey:TopicID"`
}

type Post struct {
	ID       uint   `gorm:"primaryKey"`
	TopicID  uint   `gorm:"not null"`
	AuthorID uint   `gorm:"not null"`
	Content  string `gorm:"type:text;not null"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Relations
	Topic  Topic `gorm:"foreignKey:TopicID"`
	Author User  `gorm:"foreignKey:AuthorID"`
}

// Helper methods for permissions
func (u *User) CanModerate() bool {
	return u.UserType == UserTypeModerator || u.UserType == UserTypeAdmin
}

func (u *User) IsAdmin() bool {
	return u.UserType == UserTypeAdmin
}

func (u *User) CanPost() bool {
	return u.UserType >= UserTypeUser && !u.IsBanned
}

func (u *User) CanEditPost(post *Post) bool {
	if u.IsBanned {
		return false
	}
	return u.ID == post.AuthorID || u.CanModerate()
}

func (u *User) CanDeletePost(post *Post) bool {
	if u.IsBanned {
		return false
	}
	return u.ID == post.AuthorID || u.CanModerate()
}

func (u *User) CanEditTopic(topic *Topic) bool {
	if u.IsBanned {
		return false
	}
	return u.ID == topic.AuthorID || u.CanModerate()
}

func (u *User) CanDeleteTopic(topic *Topic) bool {
	if u.IsBanned {
		return false
	}
	return u.ID == topic.AuthorID || u.CanModerate()
}

func (u *User) IsActive() bool {
	if u.IsBanned {
		if u.BannedUntil != nil && time.Now().After(*u.BannedUntil) {
			return true // Ban has expired
		}
		return false
	}
	return true
}
