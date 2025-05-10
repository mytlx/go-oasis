package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	ID   int
	Name string
	Age  int
}

func main() {
	// 1. 连接数据库（自动创建文件）
	db, err := sql.Open("sqlite3", "../database/demo.db")
	checkErr(err)
	defer db.Close()

	// 2. 创建表
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS user (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		age INTEGER
	);`
	_, err = db.Exec(createTableSQL)
	checkErr(err)

	// 3. 插入数据（Prepare + Exec）
	stmt, err := db.Prepare("INSERT INTO user (name, age) VALUES (?, ?)")
	checkErr(err)
	defer stmt.Close()
	_, err = stmt.Exec("Alice", 18)
	checkErr(err)
	_, err = stmt.Exec("Bob", 20)
	checkErr(err)

	// 4. 查询数据
	rows, err := db.Query("SELECT id, name, age FROM user")
	checkErr(err)
	defer rows.Close()

	for rows.Next() {
		var user User
		err = rows.Scan(&user.ID, &user.Name, &user.Age)
		checkErr(err)
		fmt.Printf("User: %+v\n", user)
	}

	// 5. 更新数据
	_, err = db.Exec("UPDATE user SET age = ? WHERE name = ?", 29, "Alice")
	checkErr(err)

	// 6. 删除数据
	_, err = db.Exec("DELETE FROM user WHERE name = ?", "Bob")
	checkErr(err)

	// 7. 事务处理（一次性插入多条）
	tx, err := db.Begin()
	checkErr(err)
	stmt2, err := tx.Prepare("INSERT INTO user(name, age) VALUES(?, ?)")
	checkErr(err)
	defer stmt2.Close()

	for _, name := range []string{"Tom", "Jerry", "Lucy"} {
		_, err = stmt2.Exec(name, 22)
		checkErr(err)
	}
	tx.Commit()

	fmt.Println("操作完成")
}

func checkErr(err error) {
	if err != nil {
		log.Fatal("错误:", err)
	}
}
