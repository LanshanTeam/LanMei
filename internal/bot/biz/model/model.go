package model

import "time"

type User struct {
	ID       int64  `json:"id" gorm:"primaryKey"`
	QQId     string `json:"qq_id" gorm:"type:varchar(255);unique;index"`
	Username string `json:"username" gorm:"type:varchar(255);default:'默认名称'"`
	Point    int64  `json:"point" gorm:"type:bigInt"`
}

type UserProfile struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	GroupID   string    `json:"group_id" gorm:"type:varchar(64);index:idx_profile_group_name,unique"`
	Name      string    `json:"name" gorm:"type:varchar(255);index:idx_profile_group_name,unique"`
	Summary   string    `json:"summary" gorm:"type:text"`
	Tags      string    `json:"tags" gorm:"type:text"`
	UpdatedAt time.Time `json:"updated_at"`
}
