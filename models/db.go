package models

import (
	"fmt"
	"log"

	"github.com/8treenet/gcache"
	"github.com/8treenet/gcache/option"
	"github.com/forewing/goldennum/config"
	"github.com/jinzhu/gorm"

	// database drivers
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

var (
	// Db saves database
	Db *gorm.DB
)

const (
	// defaultDbConfig = "file::memory:?cache=shared"
	defaultDbConfig = "./sqlite3.db"
)

// Load init Db from config
func Load() {
	if Db != nil {
		log.Panicln("[models] Load init twice")
	}

	conf := config.Load()

	var err error
	switch conf.Db.Type {
	case "sqlite3":
		Db, err = gorm.Open("sqlite3", conf.Db.Path)
	case "mysql":
		url := fmt.Sprintf("%v:%v@(%v)/%v?charset=utf8mb4&parseTime=True&loc=Local",
			conf.Db.User, conf.Db.Password, conf.Db.Addr, conf.Db.DbName)
		Db, err = gorm.Open("mysql", url)
	default:
		log.Println("Error: [models] Load db config not found or invalid, using ", defaultDbConfig)
		Db, err = gorm.Open("sqlite3", defaultDbConfig)
	}

	if err != nil {
		panic(err)
	}

	Db.AutoMigrate(&User{}, &UserHistory{}, &Room{}, &RoomHistory{})

	if len(conf.Db.Redis) != 0 {
		opt := option.DefaultOption{}
		opt.Level = option.LevelModel
		opt.AsyncWrite = true
		gcache.AttachDB(Db, &opt, &option.RedisOption{Addr: conf.Db.Redis})
	}
}

// Close Db
func Close() {
	Db.Close()
}
