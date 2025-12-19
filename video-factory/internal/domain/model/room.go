package model

type Room struct {
	ID           int64  `gorm:"column:id;primaryKey"`
	Platform     string `gorm:"column:platform"`
	ShortID      string `gorm:"column:short_id"`
	RealID       string `gorm:"column:real_id"`
	Name         string `gorm:"column:name"`
	URL          string `gorm:"column:url"`
	CoverURL     string `gorm:"column:cover_url"`
	ProxyURL     string `gorm:"column:proxy_url"`
	AnchorID     string `gorm:"column:anchor_id"`
	AnchorName   string `gorm:"column:anchor_name"`
	AnchorAvatar string `gorm:"column:anchor_avatar"`
	Status       int    `gorm:"column:status;not null;default:0"`        // 0: 禁用 1: 启用
	RecordStatus int    `gorm:"column:record_status;not null;default:0"` // 录制状态，0：禁用 1：启用
	CreateTime   int64  `gorm:"column:create_time;autoCreateTime:milli;type:integer"`
	UpdateTime   int64  `gorm:"column:update_time;autoUpdateTime:milli;type:integer"`
}

func (Room) TableName() string {
	return "t_room"
}
