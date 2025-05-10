package model

type User struct {
	ID   int64  `gorm:"primaryKey"`
	Name string `gorm:"size:100;not null"`
	Age  int    `gorm:"not null"`
}
