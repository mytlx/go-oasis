package model

type Room struct {
	ID         string    `gorm:"column:id;primaryKey"`
	Platform   string    `gorm:"column:platform"`
	RealID     string    `gorm:"column:real_id"`
	Name       string    `gorm:"column:name"`
	URL        string    `gorm:"column:url"`
	ProxyURL   string    `gorm:"column:proxy_url"`
	CreateTime int64     `gorm:"column:create_time;autoCreateTime:milli;type:integer"`
	UpdateTime int64     `gorm:"column:update_time;autoUpdateTime:milli;type:integer"`
}

func (Room) TableName() string {
	return "t_room"
}
