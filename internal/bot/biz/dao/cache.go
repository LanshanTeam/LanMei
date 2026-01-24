package dao

import (
	"LanMei/internal/bot/utils/llog"
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CacheManagerImpl struct {
	db *gorm.DB
}

type CacheKV struct {
	CacheKey string     `gorm:"column:cache_key;primaryKey;type:text"`
	Value    string     `gorm:"column:value;type:text"`
	ExpireAt *time.Time `gorm:"column:expire_at"`
}

func (CacheKV) TableName() string {
	return "cache_kv"
}

type CacheHash struct {
	CacheKey string `gorm:"column:cache_key;primaryKey;type:text"`
	Field    string `gorm:"column:field;primaryKey;type:text"`
	Value    int64  `gorm:"column:value;type:bigint"`
}

func (CacheHash) TableName() string {
	return "cache_hash"
}

func NewCacheManager(db *gorm.DB) *CacheManagerImpl {
	return &CacheManagerImpl{db: db}
}

func (m *CacheManagerImpl) setNX(ctx context.Context, key string, value string, ttl time.Duration) (bool, error) {
	expireAt := time.Now().Add(ttl)
	db := m.db.WithContext(ctx)
	for i := 0; i < 2; i++ {
		res := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&CacheKV{
			CacheKey: key,
			Value:    value,
			ExpireAt: &expireAt,
		})
		if res.Error != nil {
			return false, res.Error
		}
		if res.RowsAffected > 0 {
			return true, nil
		}
		now := time.Now()
		cleanup := db.Where("cache_key = ? AND expire_at IS NOT NULL AND expire_at <= ?", key, now).
			Delete(&CacheKV{})
		if cleanup.Error != nil {
			return false, cleanup.Error
		}
		if cleanup.RowsAffected == 0 {
			return false, nil
		}
	}
	return false, nil
}

func (m *CacheManagerImpl) set(ctx context.Context, key string, value string, ttl time.Duration) error {
	expireAt := time.Now().Add(ttl)
	return m.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "cache_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "expire_at"}),
	}).Create(&CacheKV{
		CacheKey: key,
		Value:    value,
		ExpireAt: &expireAt,
	}).Error
}

func (m *CacheManagerImpl) get(ctx context.Context, key string) (string, error) {
	var item CacheKV
	err := m.db.WithContext(ctx).
		Where("cache_key = ? AND (expire_at IS NULL OR expire_at > ?)", key, time.Now()).
		First(&item).Error
	if err != nil {
		return "", err
	}
	return item.Value, nil
}

func (m *CacheManagerImpl) exists(ctx context.Context, key string) (bool, error) {
	var item CacheKV
	err := m.db.WithContext(ctx).
		Select("cache_key").
		Where("cache_key = ? AND (expire_at IS NULL OR expire_at > ?)", key, time.Now()).
		Take(&item).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}
	return err == nil, nil
}

func (m *CacheManagerImpl) hincrBy(ctx context.Context, key string, field string, value int64) error {
	return m.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "cache_key"}, {Name: "field"}},
		DoUpdates: clause.Assignments(map[string]any{
			"value": gorm.Expr("cache_hash.value + EXCLUDED.value"),
		}),
	}).Create(&CacheHash{
		CacheKey: key,
		Field:    field,
		Value:    value,
	}).Error
}

func (m *CacheManagerImpl) hgetAll(ctx context.Context, key string) (map[string]int64, error) {
	var rows []CacheHash
	err := m.db.WithContext(ctx).Where("cache_key = ?", key).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, r := range rows {
		out[r.Field] = r.Value
	}
	return out, nil
}

func (m *CacheManagerImpl) hget(ctx context.Context, key string, field string) (int64, error) {
	var item CacheHash
	err := m.db.WithContext(ctx).
		Where("cache_key = ? AND field = ?", key, field).
		Take(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return item.Value, nil
}

func (m *CacheManagerImpl) hset(ctx context.Context, key string, field string, value int64) error {
	return m.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "cache_key"}, {Name: "field"}},
		DoUpdates: clause.AssignmentColumns([]string{"value"}),
	}).Create(&CacheHash{
		CacheKey: key,
		Field:    field,
		Value:    value,
	}).Error
}

func (m *DBManagerImpl) MarkAsSigned(ctx context.Context, qqId string) error {
	date := time.Now().Format("2006-01-02")
	key := fmt.Sprintf("sign:%v:%v", qqId, date)
	exists, err := m.cacheDB.setNX(ctx, key, "1", 24*time.Hour)
	llog.Info("", exists, err)
	if !exists {
		return fmt.Errorf("今日已签到")
	}
	return nil
}

func (m *DBManagerImpl) MarkDayilyLuck(ctx context.Context, qqId string, sign int) int {
	date := time.Now().Format("2006-01-02")
	key := fmt.Sprintf("daily_luck:%v:%v:", qqId, date)
	exists, err := m.cacheDB.setNX(ctx, key, strconv.Itoa(sign), 24*time.Hour)
	llog.Info("", exists, err)
	if !exists {
		str, _ := m.cacheDB.get(ctx, key)
		ans, _ := strconv.Atoi(str)
		return ans
	}
	return -1
}

func (m *DBManagerImpl) StaticWords(ctx context.Context, words map[string]int64, groupId string) {
	key := fmt.Sprintf("wordcloud:%v", groupId)
	for k, v := range words {
		err := m.cacheDB.hincrBy(ctx, key, k, v)
		if err != nil {
			llog.Error("添加词条失败！", k, v)
			continue
		}
	}
}

func (m *DBManagerImpl) GetWords(ctx context.Context, groupId string) map[string]int64 {
	key := fmt.Sprintf("wordcloud:%v", groupId)
	res, err := m.cacheDB.hgetAll(ctx, key)
	if err != nil {
		llog.Error("查询词库失败")
		return nil
	}
	return res
}

func (m *DBManagerImpl) IncrJargonCount(ctx context.Context, groupId, term string) (int64, error) {
	key := fmt.Sprintf("jargon_count:%v", groupId)
	if err := m.cacheDB.hincrBy(ctx, key, term, 1); err != nil {
		return 0, err
	}
	return m.cacheDB.hget(ctx, key, term)
}

func (m *DBManagerImpl) GetJargonLastInfer(ctx context.Context, groupId, term string) (int64, error) {
	key := fmt.Sprintf("jargon_infer:%v", groupId)
	return m.cacheDB.hget(ctx, key, term)
}

func (m *DBManagerImpl) SetJargonLastInfer(ctx context.Context, groupId, term string, value int64) error {
	key := fmt.Sprintf("jargon_infer:%v", groupId)
	return m.cacheDB.hset(ctx, key, term, value)
}

func (m *DBManagerImpl) HasJrlp(ctx context.Context, groupId int64, userId int64) bool {
	key := fmt.Sprintf("jrlp:%v:%v:%v:qq", groupId, time.Now().Format("2006-01-02"), userId)
	res, err := m.cacheDB.exists(ctx, key)
	if err != nil {
		return false
	}
	if !res {
		return false
	}
	return true
}

func (m *DBManagerImpl) SetJrlp(ctx context.Context, groupId int64, userId int64, lpId int64, msg string) {
	qqkey := fmt.Sprintf("jrlp:%v:%v:%v:qq", groupId, time.Now().Format("2006-01-02"), userId)
	err := m.cacheDB.set(ctx, qqkey, strconv.FormatInt(lpId, 10), 24*time.Hour)
	if err != nil {
		llog.Error("设置今日老婆失败！", err)
	}
	msgKey := fmt.Sprintf("jrlp:%v:%v:%v:msg", groupId, time.Now().Format("2006-01-02"), userId)
	err = m.cacheDB.set(ctx, msgKey, msg, 24*time.Hour)
	if err != nil {
		llog.Error("设置今日老婆消息失败！", err)
	}
}

func (m *DBManagerImpl) GetJrlp(ctx context.Context, groupId int64, userId int64) (int64, string) {
	qqkey := fmt.Sprintf("jrlp:%v:%v:%v:qq", groupId, time.Now().Format("2006-01-02"), userId)
	res, err := m.cacheDB.get(ctx, qqkey)
	if err != nil {
		llog.Error("获取今日老婆失败！", err)
		return 0, ""
	}
	msgKey := fmt.Sprintf("jrlp:%v:%v:%v:msg", groupId, time.Now().Format("2006-01-02"), userId)
	msg, err := m.cacheDB.get(ctx, msgKey)
	if err != nil {
		llog.Error("获取今日老婆消息失败！", err)
		return 0, ""
	}
	ans, _ := strconv.ParseInt(res, 10, 64)
	return ans, msg
}
