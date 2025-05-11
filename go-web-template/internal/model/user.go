package model

// User 建表语句
// CREATE TABLE `user` (
//
//	`id` bigint NOT NULL AUTO_INCREMENT,
//	`name` varchar(255) CHARACTER SET utf8 COLLATE utf8_general_ci DEFAULT NULL,
//	`gender` int DEFAULT NULL COMMENT '0：女，1：男',
//	`age` int DEFAULT NULL,
//	`email` varchar(255) CHARACTER SET utf8 COLLATE utf8_general_ci DEFAULT NULL,
//	`address` varchar(255) CHARACTER SET utf8 COLLATE utf8_general_ci DEFAULT NULL,
//	PRIMARY KEY (`id`)
//
// ) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;
type User struct {
	ID      int64  `gorm:"primaryKey" json:"id"`
	Name    string `gorm:"column:name" binding:"required" json:"name"`
	Gender  *int   `gorm:"column:gender" binding:"required,oneof=0 1" json:"gender"`
	Age     int    `gorm:"column:age" binding:"gte=0,lte=120" json:"age"`
	Email   string `gorm:"column:email" binding:"required,email" json:"email"`
	Address string `gorm:"column:address" json:"address"`
}

// TableName 指定表名
func (User) TableName() string {
	return "user"
}
