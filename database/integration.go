package database

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// MigrationConfig 迁移配置
type MigrationConfig struct {
	DatabaseURL    string // 数据库连接字符串
	MigrationsPath string // 迁移文件路径
	AutoMigrate    bool   // 是否自动执行迁移
}

// MigrationManager 迁移管理器
type MigrationManager struct {
	config *MigrationConfig
	db     *sql.DB
}

// NewMigrationManager 创建迁移管理器
func NewMigrationManager(db *sql.DB, config *MigrationConfig) *MigrationManager {
	if config.MigrationsPath == "" {
		// 默认使用相对于当前文件的迁移路径
		_, currentFile, _, _ := runtime.Caller(0)
		config.MigrationsPath = filepath.Join(filepath.Dir(currentFile), "migrations")
	}

	return &MigrationManager{
		config: config,
		db:     db,
	}
}

// RunMigrations 执行数据库迁移
func (m *MigrationManager) RunMigrations() error {
	driver, err := mysql.WithInstance(m.db, &mysql.Config{})
	if err != nil {
		return fmt.Errorf("could not create mysql driver: %w", err)
	}

	migrator, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", m.config.MigrationsPath),
		"mysql",
		driver,
	)
	if err != nil {
		return fmt.Errorf("could not create migrate instance: %w", err)
	}
	defer migrator.Close()

	if err := migrator.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("could not run migrations: %w", err)
	}

	return nil
}

// GetVersion 获取当前迁移版本
func (m *MigrationManager) GetVersion() (uint, bool, error) {
	driver, err := mysql.WithInstance(m.db, &mysql.Config{})
	if err != nil {
		return 0, false, fmt.Errorf("could not create mysql driver: %w", err)
	}

	migrator, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", m.config.MigrationsPath),
		"mysql",
		driver,
	)
	if err != nil {
		return 0, false, fmt.Errorf("could not create migrate instance: %w", err)
	}
	defer migrator.Close()

	return migrator.Version()
}

// MigrateTo 迁移到指定版本
func (m *MigrationManager) MigrateTo(version uint) error {
	driver, err := mysql.WithInstance(m.db, &mysql.Config{})
	if err != nil {
		return fmt.Errorf("could not create mysql driver: %w", err)
	}

	migrator, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", m.config.MigrationsPath),
		"mysql",
		driver,
	)
	if err != nil {
		return fmt.Errorf("could not create migrate instance: %w", err)
	}
	defer migrator.Close()

	if err := migrator.Migrate(version); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("could not migrate to version %d: %w", version, err)
	}

	return nil
}

// RollbackSteps 回滚指定步数
func (m *MigrationManager) RollbackSteps(steps int) error {
	driver, err := mysql.WithInstance(m.db, &mysql.Config{})
	if err != nil {
		return fmt.Errorf("could not create mysql driver: %w", err)
	}

	migrator, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", m.config.MigrationsPath),
		"mysql",
		driver,
	)
	if err != nil {
		return fmt.Errorf("could not create migrate instance: %w", err)
	}
	defer migrator.Close()

	if err := migrator.Steps(-steps); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("could not rollback %d steps: %w", steps, err)
	}

	return nil
}

// DefaultMigrationConfig 返回默认迁移配置
func DefaultMigrationConfig() *MigrationConfig {
	return &MigrationConfig{
		AutoMigrate: false, // 默认不自动迁移，在生产环境中应该手动控制
	}
}

// Example 使用示例
func Example() {
	// 1. 创建数据库连接
	// db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/dbname?parseTime=true")
	// if err != nil {
	//     log.Fatal(err)
	// }
	// defer db.Close()

	// 2. 创建迁移管理器
	// config := DefaultMigrationConfig()
	// config.MigrationsPath = "./database/migrations"
	// migrator := NewMigrationManager(db, config)

	// 3. 执行迁移
	// if err := migrator.RunMigrations(); err != nil {
	//     log.Fatalf("Migration failed: %v", err)
	// }

	// 4. 检查版本
	// version, dirty, err := migrator.GetVersion()
	// if err != nil {
	//     log.Printf("Could not get version: %v", err)
	// } else {
	//     log.Printf("Current migration version: %d, dirty: %t", version, dirty)
	// }
}