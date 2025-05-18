package dao

import (
	"local-transfer/internal/db"
	"local-transfer/internal/model"
	"local-transfer/internal/vo"
)

func CreateMessage(message *model.Message) error {
	return db.DB.Create(message).Error
}

func GetMessagesBySrcAndDst(vo *vo.MessageQueryVO) ([]model.Message, error) {
	srcId := vo.SrcDeviceId
	dstId := vo.DstDeviceId
	beforeId := vo.BeforeId

	var messages []model.Message
	// err := db.DB.Where("(source_id = ? AND target_id = ?) OR (source_id = ? AND target_id = ?)",
	// 	srcId, dstId, dstId, srcId).Order("create_time ASC").Find(&messages).Error

	query := db.DB.Where(
		"(source_id = ? AND target_id = ?) OR (source_id = ? AND target_id = ?)",
		srcId, dstId, dstId, srcId,
	)
	if beforeId > 0 {
		query = query.Where("id < ?", beforeId)
	}

	err := query.Order("id DESC").Limit(vo.Limit).Find(&messages).Error

	return messages, err
}
