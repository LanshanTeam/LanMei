package dao

import (
	"context"
	"errors"
	"time"

	"LanMei/internal/bot/biz/model"

	"gorm.io/gorm"
)

func (m *DBManagerImpl) GetUserProfile(ctx context.Context, groupID, name string) (*model.UserProfile, error) {
	if m == nil || m.db == nil {
		return nil, nil
	}
	var profile model.UserProfile
	err := m.db.db.WithContext(ctx).Where("group_id = ? AND name = ?", groupID, name).First(&profile).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (m *DBManagerImpl) UpsertUserProfile(ctx context.Context, profile *model.UserProfile) error {
	if m == nil || m.db == nil || profile == nil {
		return nil
	}
	if profile.GroupID == "" || profile.Name == "" {
		return nil
	}
	db := m.db.db.WithContext(ctx)
	var existing model.UserProfile
	err := db.Where("group_id = ? AND name = ?", profile.GroupID, profile.Name).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if profile.ID == 0 && SFNode != nil {
			profile.ID = SFNode.Generate().Int64()
		}
		profile.UpdatedAt = time.Now()
		return db.Create(profile).Error
	}
	if err != nil {
		return err
	}
	return db.Model(&existing).Updates(map[string]any{
		"summary":    profile.Summary,
		"tags":       profile.Tags,
		"updated_at": time.Now(),
	}).Error
}
