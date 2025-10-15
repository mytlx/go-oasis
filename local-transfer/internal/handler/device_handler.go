package handler

import (
	"github.com/gin-gonic/gin"
	"local-transfer/internal/model"
	"local-transfer/internal/ws"
	"net/http"
	"strconv"
)

func GetDeviceByIdHandler(c *gin.Context) {
	deviceIdStr := c.Param("id")
	if deviceIdStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
	}
	deviceId, err := strconv.ParseInt(deviceIdStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is valid"})
		return
	}
	client := ws.GetDeviceClient(deviceId)
	if client == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": &model.Device{
			ID:   deviceId,
			IP:   client.IP,
			Name: client.DeviceName,
			Type: client.DeviceType,
		},
	})

}
