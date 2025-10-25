package model

type Config struct {
	ID          int64  `gorm:"column:id;primaryKey"`
	Key         string `gorm:"column:key"`
	Value       string `gorm:"column:value"`
	Description string `gorm:"column:description"`
	CreateTime  int64  `gorm:"column:create_time;autoCreateTime:milli;type:integer"`
	UpdateTime  int64  `gorm:"column:update_time;autoUpdateTime:milli;type:integer"`
}

func (Config) TableName() string {
	return "t_config"
}
