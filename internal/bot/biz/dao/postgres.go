package dao

import (
	"LanMei/internal/bot/biz/model"
	"LanMei/internal/bot/config"
	"LanMei/internal/bot/utils/llog"
	"errors"
	"fmt"

	"github.com/bwmarrin/snowflake"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DBManagerImpl struct {
	db      *PostgresManagerImpl
	cacheDB *CacheManagerImpl
	embedDB *EmbeddingManagerImpl
}

type PostgresManagerImpl struct {
	db *gorm.DB
}

var (
	DBManager *DBManagerImpl
	SFNode    *snowflake.Node
)

// InitDBManager 初始化 dao 层的 manager
func InitDBManager() {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		config.K.String("Database.Postgres.Host"),
		config.K.String("Database.Postgres.User"),
		config.K.String("Database.Postgres.Password"),
		config.K.String("Database.Postgres.DBName"),
		config.K.String("Database.Postgres.Port"),
		config.K.String("Database.Postgres.SSLMode"),
		config.K.String("Database.Postgres.TimeZone"),
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		llog.Fatal("open db error: ", err)
	}
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
		llog.Fatal("failed to enable pgvector extension: ", err)
	}

	// 设置连接池参数
	sqlDB, err := db.DB()
	if err != nil {
		llog.Fatal("failed to get sql.DB from GORM: ", err)
		return
	}
	sqlDB.SetMaxOpenConns(50) // 最大连接数
	DBManager = &DBManagerImpl{
		db:      NewPostgresManager(db),
		cacheDB: NewCacheManager(db),
		embedDB: NewEmbeddingManager(db),
	}
	db.AutoMigrate(&model.User{}, &CacheKV{}, &CacheHash{}, &EmbeddingRecord{})
}

func NewPostgresManager(db *gorm.DB) *PostgresManagerImpl {
	return &PostgresManagerImpl{
		db: db,
	}
}

func InitSnowFlakeNode() {
	var err error
	// 创建雪花算法节点
	SFNode, err = snowflake.NewNode(0)
	if err != nil {
		llog.Fatal("Create SnowflakeNode Error: " + err.Error())
	}
}

func (m *DBManagerImpl) GetUserDefine(user *model.User) error {
	return m.db.db.Where("qq_id = ?", user.QQId).First(&user).Error
}

func (m *DBManagerImpl) AddPoint(user *model.User, point int) error {
	err := m.db.db.Where("qq_id = ?", user.QQId).First(&user).Error

	// 先看看有没有这个用户
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 雪花算法生成 id
		id := SFNode.Generate().Int64()
		user := model.User{
			ID:       id,
			QQId:     user.QQId,
			Username: user.Username,
			Point:    int64(point),
		}
		res := m.db.db.Create(&user)
		if res.Error != nil {
			llog.Error("插入用户数据失败：", res.Error)
			return res.Error
		}
		return nil
	}

	// 已存在，加分
	res := m.db.db.Model(&user).Updates(map[string]any{
		"point":    gorm.Expr("point + ?", point),
		"username": user.Username,
	})
	if res.Error != nil {
		llog.Error("更新失败", err)
		return res.Error
	}
	return nil
}

func (m *DBManagerImpl) GetUserRank(user *model.User) (int, error) {
	var rank int64
	err := m.db.db.Model(user).
		Where("point > (?)", m.db.db.Model(user).Select("point").Where("qq_id = ?", user.QQId)).
		Count(&rank).Error
	if err != nil {
		llog.Error("查询用户排名失败: %v", err)
		return 0, err
	}
	// 排名 = 比我分数高的人数 + 1
	return int(rank) + 1, nil
}

func (m *DBManagerImpl) RankList() ([]model.User, error) {
	var users []model.User
	err := m.db.db.Model(&model.User{}).
		Select("id, qq_id, username, point").
		Order("point DESC").
		Limit(10).
		Find(&users).Error

	if err != nil {
		llog.Error("查询排行榜失败: %v", err)
		return nil, err
	}
	return users, nil
}

func (m *DBManagerImpl) SetName(qqId string, username string) error {
	var user model.User
	err := m.db.db.Where("qq_id = ?", qqId).First(&user).Error

	// 先看看有没有这个用户
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 雪花算法生成 id
		id := SFNode.Generate().Int64()
		user := model.User{
			ID:       id,
			QQId:     qqId,
			Username: username,
			Point:    0,
		}
		res := m.db.db.Create(user)
		if res.Error != nil {
			llog.Error("插入用户数据失败：", res.Error)
			return res.Error
		}
		return nil
	}

	err = m.db.db.Model(&user).Where("qq_id = ?", qqId).Update("username", username).Error
	if err != nil {
		llog.Error("更新失败：", err)
		return err
	}
	return nil
}
