package db

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

var DB *sql.DB

func InitDB(dbpath string) {
	var err error
	DB, err = sql.Open("sqlite3", dbpath)
	if err != nil {
		log.Fatal("打开数据库失败:", err)
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS user (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		age INTEGER
	);`
	_, err = DB.Exec(createTableSQL)
	if err != nil {
		log.Fatal("建表失败:", err)
	}
}

// HandleError 通用的错误处理方法
func HandleError(err error, format string, args ...any) {
	if err != nil {
		log.Printf("错误: "+format+",\n错误信息: %v\n", append(args, err)...)
	}
}
