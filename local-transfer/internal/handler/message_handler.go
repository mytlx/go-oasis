package handler

import (
	"github.com/gin-gonic/gin"
	"local-transfer/internal/service"
	"local-transfer/internal/vo"
	"net/http"
)

func ListMsgHandler(c *gin.Context) {

	var queryVO vo.MessageQueryVO
	if err := c.ShouldBindQuery(&queryVO); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// sourceIdStr := c.Query("srcDeviceId")
	// sourceId, _ := strconv.ParseInt(sourceIdStr, 10, 64)
	// if sourceId == 0 {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "source_id is required"})
	// }
	//
	// targetIdStr := c.Query("dstDeviceId")
	// targetId, _ := strconv.ParseInt(targetIdStr, 10, 64)
	// if targetId == 0 {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "target_id is required"})
	// }
	//
	// // 获取 beforeId 和 limit，尝试转成整数
	// beforeIdStr := c.DefaultQuery("beforeId", "0")
	// beforeId, _ := strconv.ParseInt(beforeIdStr, 10, 64) // 可以加 error 处理
	//
	// limitStr := c.DefaultQuery("limit", "20")
	// limit, _ := strconv.Atoi(limitStr)

	messageVOS, err := service.GetMessagesBySrcAndDst(&queryVO)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"messages": messageVOS})
}
