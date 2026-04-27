package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// Fail 失败响应
func Fail(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
	})
}

// Error 错误响应
func Error(c *gin.Context, err error) {
	c.JSON(http.StatusOK, Response{
		Code:    500,
		Message: err.Error(),
	})
}

// OkWithData 返回数据
func OkWithData(data interface{}, c *gin.Context) {
	Success(c, data)
}

// OkWithMessage 返回消息
func OkWithMessage(message string, c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: message,
	})
}

// FailWithMessage 失败消息
func FailWithMessage(message string, c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Code:    1,
		Message: message,
	})
}

// ParamError 参数错误
func ParamError(c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Code:    400,
		Message: "参数错误",
	})
}
