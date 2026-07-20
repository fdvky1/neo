package database

import "gorm.io/gorm"

type Filter struct {
	gorm.Model
	Key     string `gorm:"type:varchar(255);uniqueIndex:idx_group_key"`
	GroupID string `gorm:"type:varchar(255);uniqueIndex:idx_group_key"`
	Content string `gorm:"type:text"`
}

func AutoMigrate() {
	DB.AutoMigrate(&Filter{})
}
