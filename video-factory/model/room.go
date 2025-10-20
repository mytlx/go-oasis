package model

// type UnixTime time.Time
//
// func (t *UnixTime) Scan(value any) error {
// 	switch v := value.(type) {
// 	case int64:
// 		*t = UnixTime(time.Unix(v, 0))
// 		return nil
// 	case int:
// 		*t = UnixTime(time.Unix(int64(v), 0))
// 		return nil
// 	default:
// 		return fmt.Errorf("cannot scan %T into UnixTime", value)
// 	}
// }
//
// func (t *UnixTime) Value() (driver.Value, error) {
// 	return time.Time(*t).Unix(), nil
// }

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
