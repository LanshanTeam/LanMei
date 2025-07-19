package model

type User struct {
	ID       int64  `json:"id" gorm:"primaryKey"`
	QQId     string `json:"qq_id" gorm:"type:varchar(255);unique;index"`
	Username string `json:"username" gorm:"type:varchar(255);default:'默认名称'"`
	Point    int64  `json:"point" gorm:"type:bigInt"`
}
