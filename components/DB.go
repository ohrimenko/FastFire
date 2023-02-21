package components

import (
	"errors"
	"fmt"
	"log"
	"os"

	"backnet/config"
	"sync"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"gorm.io/driver/sqlite"
	// Pure go SQLite driver, checkout https://github.com/glebarez/sqlite for details.
	// Преимущество в том что это чистый GO без gcc
	// Основной не достаток что медленее.
	// "github.com/glebarez/sqlite"
)

type dbStruct struct {
	connect *gorm.DB
	err     error
	valid   bool

	mutex sync.Mutex
}

var dbApp dbStruct

func (n *dbStruct) db() (*gorm.DB, error) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if !n.valid {
		if config.Env("DB_DRIVER") == "mysql" {
			dsn := config.Env("DB_USERNAME") + ":" + config.Env("DB_PASSWORD") + "@tcp(" + config.Env("DB_HOST") + ":" + config.Env("DB_PORT") + ")/" + config.Env("DB_DATABASE") + "?charset=" + config.Env("DB_CHARSET") + "&parseTime=True&loc=Local"

			n.connect, n.err = gorm.Open(mysql.New(mysql.Config{
				DSN:                       dsn,   // data source name
				DefaultStringSize:         256,   // default size for string fields
				DisableDatetimePrecision:  true,  // disable datetime precision, which not supported before MySQL 5.6
				DontSupportRenameIndex:    true,  // drop & create when rename index, rename index not supported before MySQL 5.7, MariaDB
				DontSupportRenameColumn:   true,  // `change` when rename column, rename column not supported before MySQL 8, MariaDB
				SkipInitializeWithVersion: false, // auto configure based on currently MySQL version
			}), &gorm.Config{})

			if n.err == nil {
				sqlDB, err := n.connect.DB()

				if err == nil {
					// SetMaxIdleConns устанавливает максимальное количество соединений в пуле незанятых соединений.
					sqlDB.SetMaxIdleConns(100)

					// SetMaxOpenConns устанавливает максимальное количество открытых подключений к базе данных.
					sqlDB.SetMaxOpenConns(0)

					// SetConnMaxLifetime устанавливает максимальное количество времени, в течение которого соединение может быть повторно использовано.
					sqlDB.SetConnMaxLifetime(0)

					sqlDB.SetConnMaxIdleTime(0)
				}
			}

			n.valid = true
		} else if config.Env("DB_DRIVER") == "sqlite" {
			n.connect, n.err = gorm.Open(sqlite.Open(config.Env("DB_FILE")), &gorm.Config{})

			if n.err == nil {
				sqlDB, err := n.connect.DB()

				if err == nil {
					// SetMaxIdleConns устанавливает максимальное количество соединений в пуле незанятых соединений.
					sqlDB.SetMaxIdleConns(100)

					// SetMaxOpenConns устанавливает максимальное количество открытых подключений к базе данных.
					sqlDB.SetMaxOpenConns(0)

					// SetConnMaxLifetime устанавливает максимальное количество времени, в течение которого соединение может быть повторно использовано.
					sqlDB.SetConnMaxLifetime(0)

					sqlDB.SetConnMaxIdleTime(0)

					_, err := n.connect.Raw("SELECT * FROM users LIMIT 1").Rows()

					if err != nil {
						body, err := os.ReadFile("storage/migrations/db.sql")

						if err == nil {
							sqlstring := ""

							СonvertAssign(&sqlstring, body)

							fmt.Println("Exec DB migration: storage/migrations/db.sql")

							n.connect.Exec(sqlstring)
						}
					}
				}
			}

			n.valid = true
		} else {
			n.err = errors.New("Error Select Driver DB")
		}
	}

	return n.connect, n.err
}

func DB() (*gorm.DB, error) {
	return dbApp.db()
}

func CloseDB() {
	DB, err := DB()

	if err != nil {
		log.Fatal(err)
	}

	sqlDB, errDB := DB.DB()

	if errDB == nil {
		sqlDB.Close()
	}
}
