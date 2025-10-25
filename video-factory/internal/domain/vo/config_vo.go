package vo

import "time"

type ConfigVO struct {
	ID          int64     `json:"id,string"`
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description string    `json:"description"`
	CreateTime  time.Time `json:"create_time"`
	UpdateTime  time.Time `json:"update_time"`
}
