package vo

import "time"

type RoomVO struct {
	ID         string    `json:"id"`
	RealID     string    `json:"realId"`
	Name       string    `json:"name"`
	Status     int       `json:"status"`
	ProxyURL   string    `json:"proxyUrl"`
	URL        string    `json:"Url"`
	Platform   string    `json:"platform"`
	CreateTime time.Time `json:"createTime"`
	UpdateTime time.Time `json:"updateTime"`
}
