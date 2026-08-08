package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/example/adnova/internal/config"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDatabase(cfg config.DatabaseConfig) (*gorm.DB, error) {
	db, err := OpenDatabase(cfg)
	if err != nil {
		return nil, err
	}
	if err := applyMigrations(db, cfg.MigrationDir); err != nil {
		return nil, err
	}
	return db, nil
}

func OpenDatabase(cfg config.DatabaseConfig) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		return nil, fmt.Errorf("connect mysql: %w", err)
	}
	return db, nil
}

type migrationRecord struct {
	Version  string `gorm:"primaryKey;size:255"`
	Checksum string `gorm:"size:64;not null"`
}

func (migrationRecord) TableName() string { return "schema_migrations" }

func applyMigrations(db *gorm.DB, directory string) error {
	if directory == "" {
		return fmt.Errorf("database migration directory must not be empty")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("read migrations from %s: %w", directory, err)
	}
	files := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)
	if len(files) == 0 || files[0] != "000000_schema_migrations.up.sql" {
		return fmt.Errorf("first migration must be 000000_schema_migrations.up.sql")
	}
	bootstrapBody, err := os.ReadFile(filepath.Join(directory, files[0]))
	if err != nil {
		return fmt.Errorf("read migration bootstrap: %w", err)
	}
	if err := executeMigrationStatements(db, files[0], bootstrapBody); err != nil {
		return err
	}
	for _, name := range files {
		body, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		digest := sha256.Sum256(body)
		checksum := hex.EncodeToString(digest[:])
		var existing migrationRecord
		result := db.Where("version = ?", name).Limit(1).Find(&existing)
		if result.Error != nil {
			return fmt.Errorf("read migration history for %s: %w", name, result.Error)
		}
		if result.RowsAffected == 1 {
			if existing.Checksum != checksum {
				return fmt.Errorf("published migration %s checksum changed", name)
			}
			continue
		}
		if err := executeMigrationStatements(db, name, body); err != nil {
			return err
		}
		if err := db.Create(&migrationRecord{Version: name, Checksum: checksum}).Error; err != nil {
			return fmt.Errorf("record migration %s: %w", name, err)
		}
	}
	return nil
}

func executeMigrationStatements(db *gorm.DB, name string, body []byte) error {
	for _, statement := range strings.Split(string(body), ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}
	return nil
}

func NewRedis(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{Addr: cfg.Address, Password: cfg.Password, DB: cfg.DB})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("connect redis: %w", err)
	}
	return client, nil
}
