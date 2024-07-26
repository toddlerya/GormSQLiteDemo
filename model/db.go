package model

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/glebarez/sqlite"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var WriteDB *gorm.DB
var ReadDB *gorm.DB

// 连接数据库
func connectDB() {

	databasePath := filepath.Join("database", "test.db")
	sqliteDSN := databasePath + `?_pragma=busy_timeout(5000)&
	_pragma=journal_mode(WAL)&
	_pragma=synchronous(1)&
	_pragma=foreign_keys(1)&
	_pragma=cache_size(1000000000)&
	_pragma=temp_store(memory)&
	_pragma=synchronous(NORMAL)&
	_txlock=immediate`

	writeDB, err := gorm.Open(sqlite.Open(sqliteDSN), &gorm.Config{})
	if err != nil {
		logrus.Errorf("建立writeDB连接失败: %s", err.Error())
		os.Exit(-1)
	}
	writeDBPool, err := writeDB.DB()
	if err != nil {
		logrus.Errorf("获取writeDB连接池失败: %s", err.Error())
		os.Exit(-1)
	}
	writeDBPool.SetMaxOpenConns(1)
	WriteDB = writeDB

	readDB, err := gorm.Open(sqlite.Open(sqliteDSN+"&_pragma=query_only(1)"), &gorm.Config{})
	if err != nil {
		logrus.Errorf("建立readDB连接失败: %s", err.Error())
		os.Exit(-1)
	}
	readDBPool, err := readDB.DB()
	if err != nil {
		logrus.Errorf("获取readDB连接池失败: %s", err.Error())
		os.Exit(-1)
	}
	readDBPool.SetMaxOpenConns(max(4, runtime.NumCPU()))
	ReadDB = readDB
}

// 自动迁移表
func migration() {
	tables := []interface{}{&Student{}, &Teacher{}}
	// 先删除已有的表
	err := WriteDB.Migrator().DropTable(tables...)
	if err != nil {
		logrus.Errorf("删除已存在的表失败: %s", err.Error())
		os.Exit(-1)
	}
	// 再重新创建
	err = WriteDB.AutoMigrate(tables...)
	if err != nil {
		logrus.Errorf("自动迁移数据库表结构失败: %s", err.Error())
		os.Exit(-1)
	}
}

// 初始化数据库
func InitDB(reMigration bool) {
	connectDB()
	if reMigration {
		migration()
	}
}
