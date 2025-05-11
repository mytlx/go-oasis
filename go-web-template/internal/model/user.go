package model

type User struct {
	ID      int64  `gorm:"primaryKey" json:"id"`
	Name    string `gorm:"column:name" binding:"required" json:"name"`
	Gender  *int   `gorm:"column:gender" binding:"required,oneof=0 1" json:"gender"`
	Age     int    `gorm:"column:age" binding:"gte=0,lte=120" json:"age"`
	Email   string `gorm:"column:email" binding:"required,email" json:"email"`
	Address string `gorm:"column:address" json:"address"`
}
