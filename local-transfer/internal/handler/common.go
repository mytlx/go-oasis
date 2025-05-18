package handler

import (
	"github.com/gin-gonic/gin"
	"local-transfer/pkg/utils"
	"net"
	"net/http"
	"strings"
)

func GetNextIdHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"nextId": utils.MustNextIDString(),
	})
}

func GetClientIPHandler(c *gin.Context) {
	forwarded := c.GetHeader("X-Forwarded-For")
	var ip string
	if forwarded != "" {
		// 多个 IP 时取第一个
		ip = strings.Split(forwarded, ",")[0]
	} else {
		ip, _, _ = net.SplitHostPort(c.Request.RemoteAddr)
	}
	c.JSON(http.StatusOK, gin.H{
		"ip": ip,
	})

}
