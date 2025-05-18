package service

import (
	"local-transfer/internal/dao"
	"local-transfer/internal/model"
	"local-transfer/internal/vo"
)

func InsertMessage(message *model.Message) {
	dao.CreateMessage(message)
}

func GetMessagesBySrcAndDst(queryVO *vo.MessageQueryVO) ([]vo.MessageVO, error) {
	messages, err := dao.GetMessagesBySrcAndDst(queryVO)
	if err != nil {
		return nil, err
	}

	messageVOs := make([]vo.MessageVO, 0)
	for _, message := range messages {
		messageVOs = append(messageVOs, vo.MessageVO{
			ID:         message.ID,
			Content:    message.Content,
			Source:     model.Device{ID: message.SourceId},
			Target:     model.Device{ID: message.TargetId},
			Type:       message.Type,
			Status:     message.Status,
			CreateTime: message.CreateTime,
		})
	}

	return messageVOs, err
}
