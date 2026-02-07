package dao

import (
	"context"
	"errors"
	"strings"
	"time"

	"LanMei/internal/bot/biz/model"

	"gorm.io/gorm"
)

func (m *DBManagerImpl) GetUserProfile(ctx context.Context, groupID, qqid string) (*model.UserProfile, error) {
	if m == nil || m.db == nil {
		return nil, nil
	}
	if strings.TrimSpace(groupID) == "" || strings.TrimSpace(qqid) == "" {
		return nil, nil
	}
	var profile model.UserProfile
	err := m.db.db.WithContext(ctx).Where("group_id = ? AND qq_id = ?", groupID, qqid).First(&profile).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		profile = model.UserProfile{
			GroupID:   groupID,
			QQID:      qqid,
			UpdatedAt: time.Now(),
		}
		if profile.ID == 0 && SFNode != nil {
			profile.ID = SFNode.Generate().Int64()
		}
		if createErr := m.db.db.WithContext(ctx).Create(&profile).Error; createErr != nil {
			return nil, createErr
		}
		return &profile, nil
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
	if profile.GroupID == "" || profile.QQID == "" {
		return nil
	}
	db := m.db.db.WithContext(ctx)
	var existing model.UserProfile
	err := db.Where("group_id = ? AND qq_id = ?", profile.GroupID, profile.QQID).First(&existing).Error
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
	updates := map[string]any{
		"summary":    profile.Summary,
		"tags":       profile.Tags,
		"updated_at": time.Now(),
	}
	if strings.TrimSpace(profile.Nickname) != "" {
		updates["nickname"] = profile.Nickname
	}
	return db.Model(&existing).Updates(updates).Error
}
