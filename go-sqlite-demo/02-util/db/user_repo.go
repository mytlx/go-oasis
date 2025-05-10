package db

import (
	"02-util/model"
)

func InsertUser(name string, age int) (int64, error) {
	result, err := DB.Exec("insert into user (name, age) values (?, ?)", name, age)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetAllUsers 查询所有用户
func GetAllUsers() ([]model.User, error) {
	rows, err := DB.Query("SELECT id, name, age FROM user")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		err := rows.Scan(&u.ID, &u.Name, &u.Age)
		if err != nil {
			HandleError(err, "扫描失败:")
			continue
		}
		users = append(users, u)
	}
	return users, nil
}

// GetUsersWithPagination 分页查询所有用户
func GetUsersWithPagination(page int, pageSize int) ([]model.User, error) {
	offset := (page - 1) * pageSize
	rows, err := DB.Query("SELECT id, name, age FROM user LIMIT ? OFFSET ?", pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var user model.User
		err := rows.Scan(&user.ID, &user.Name, &user.Age)
		if err != nil {
			HandleError(err, "扫描失败:")
			continue
		}
		users = append(users, user)
	}

	return users, nil
}

func GetById(id int64) (*model.User, error) {
	rows, err := DB.Query("SELECT id, name, age FROM user WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var user model.User
		err := rows.Scan(&user.ID, &user.Name, &user.Age)
		if err != nil {
			HandleError(err, "查询失败，id：%d\n", id)
			return nil, err
		}
		return &user, nil
	}
	// 用户不存在，返回 nil
	return nil, nil
}

// UpdateUser 更新用户
func UpdateUser(id int64, name string, age int) error {
	_, err := DB.Exec("UPDATE user SET name=?, age=? WHERE id=?", name, age, id)
	return err
}

// DeleteUser 删除用户
func DeleteUser(id int64) error {
	_, err := DB.Exec("DELETE FROM user WHERE id=?", id)
	return err
}
