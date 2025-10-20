package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 是通用的 API 响应结构体
type Response struct {
	Code    int         `json:"code"`    // 业务状态码 (自定义，如 0 代表成功)
	Data    interface{} `json:"db"`    // 响应数据主体，使用 interface{} 适应任何类型
	Message string      `json:"message"` // 状态信息或错误信息
}

// PagingData 是用于返回列表数据的结构体
type PagingData struct {
	// List: 实际的数据列表，可以是任何切片类型
	List interface{} `json:"list"`
	// Total: 数据总条数 (用于前端分页)
	Total int64 `json:"total"`
	// Page: 当前页码 (可选)
	Page int `json:"page"`
	// PageSize: 每页大小 (可选)
	PageSize int `json:"pageSize"`
}

// CodeSuccess 通用成功响应的业务状态码
const CodeSuccess = 0

func Success(c *gin.Context, data interface{}, msg string) {
	if msg == "" {
		msg = "操作成功"
	}
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: msg,
		Data:    data,
	})
}

func Fail(c *gin.Context, code int, msg string) {
	if code == CodeSuccess {
		code = 500 // 避免业务错误码和成功码冲突
	}
	if msg == "" {
		msg = "failed"
	}
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: msg,
		Data:    nil,
	})
}

// ----------------------------------------------------

// Ok 快捷成功响应，不带数据
func Ok(c *gin.Context) {
	Success(c, nil, "success")
}

func OkWithMsg(c *gin.Context, msg string) {
	Success(c, nil, msg)
}

// OkWithData 快捷成功响应，带数据
func OkWithData(c *gin.Context, data interface{}) {
	Success(c, data, "success")
}

// OkWithList 封装了成功响应，并包含列表数据和分页信息
func OkWithList(c *gin.Context, list interface{}, total int64, page int, pageSize int) {
	Success(c, PagingData{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, "success")
}

// Error 快捷失败响应
func Error(c *gin.Context, msg string) {
	Fail(c, 500, msg)
}
