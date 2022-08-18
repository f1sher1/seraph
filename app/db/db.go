package db

import (
	"errors"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/url"
	"sync"
	"time"
)

var (
	dbMu   sync.Mutex
	dbConn *gorm.DB
	config *Config
)

type Config struct {
	Connection  string
	Debug       bool
	PoolSize    int
	IdleTimeout int
}

func Init(cfg *Config) error {
	dbMu.Lock()
	defer dbMu.Unlock()

	if dbConn != nil {
		return nil
	}

	if cfg.PoolSize == 0 {
		cfg.PoolSize = 5
	}
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = 3600
	}

	uri, err := url.Parse(cfg.Connection)
	if err != nil {
		return err
	}

	var dialector gorm.Dialector
	switch uri.Scheme {
	case "sqlite":
		dialector = sqlite.Open(uri.Path)
	case "mysql":
		connStr := fmt.Sprintf("%s@tcp(%s)%s?%s", uri.User.String(), uri.Host, uri.Path, uri.RawQuery)
		dialector = mysql.Open(connStr)
	}

	if dialector == nil {
		return errors.New(fmt.Sprintf("dialector '%s' is not supported", cfg.Connection))
	}

	dbConn, err = gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return err
	}
	if cfg.Debug {
		dbConn = dbConn.Debug()
	}

	sqlDB, err := dbConn.DB()
	if err != nil {
		return nil
	}

	sqlDB.SetMaxOpenConns(cfg.PoolSize)
	sqlDB.SetConnMaxIdleTime(time.Duration(cfg.IdleTimeout) * time.Second)
	config = cfg
	return nil
}

func GetDBConnection() *gorm.DB {
	return dbConn
}
