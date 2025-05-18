package model

type Message struct {
	ID         int64       `json:"id" gorm:"primaryKey;not null"`         // 消息ID
	SourceId   int64       `json:"source_id" gorm:"not null"`             // 来源设备ID
	TargetId   int64       `json:"target_id" gorm:"not null"`             // 目标设备ID
	Type       MessageType `json:"type" gorm:"type:text;not null"`        // 消息类型：text | image | file
	Content    string      `json:"content" gorm:"type:text"`              // 文本内容或文件名
	Status     int8        `json:"status" gorm:"not null"`                // 消息状态，暂时无用
	CreateTime string      `json:"create_time" gorm:"type:text;not null"` // ISO 时间（如 2025-05-17T12:34:56Z）
}

func (Message) TableName() string {
	return "t_message"
}

type MessageType string

const (
	MessageTypeRegister MessageType = "register"
	MessageTypeText     MessageType = "text"
	MessageTypeImage    MessageType = "image"
	MessageTypeFile     MessageType = "file"
)

func (m MessageType) isValid() bool {
	switch m {
	case MessageTypeRegister, MessageTypeText, MessageTypeImage, MessageTypeFile:
		return true
	default:
		return false
	}
}
