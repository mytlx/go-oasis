package vo

import "local-transfer/internal/model"

type MessageVO struct {
	ID         int64             `json:"id,string"`
	Content    string            `json:"content"`
	Source     model.Device      `json:"source"`
	Target     model.Device      `json:"target"`
	Type       model.MessageType `json:"type"`
	Status     int8              `json:"status"`
	CreateTime string            `json:"create_time"`
}

type MessageQueryVO struct {
	SrcDeviceId int64 `form:"srcDeviceId" binding:"required"`
	DstDeviceId int64 `form:"dstDeviceId" binding:"required"`
	BeforeId    int64 `form:"beforeId,default=0"`
	Limit       int   `form:"limit,default=20"`
}
