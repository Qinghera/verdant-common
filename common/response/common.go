package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// ErrorCode 错误码
type ErrorCode int

// 错误码定义
const (
	Success       ErrorCode = 0
	ParamError    ErrorCode = 1
	NetworkError  ErrorCode = 2
	NotFoundError ErrorCode = 404
	SystemError   ErrorCode = 7
	Unauthorized  ErrorCode = 401
	Forbidden     ErrorCode = 403
	IdNotEmpty    ErrorCode = 50001
	RateLimited   ErrorCode = 429
	Timeout       ErrorCode = 408
)

// 错误码消息映射
var errorCodeMessages = map[ErrorCode]string{
	Success:       "操作成功",
	ParamError:    "参数错误",
	NetworkError:  "网络错误",
	NotFoundError: "资源不存在",
	SystemError:   "系统错误",
	Unauthorized:  "未授权",
	Forbidden:     "禁止访问",
	IdNotEmpty:    "ID不能为空",
	RateLimited:   "请求过于频繁",
	Timeout:       "请求超时",
}

// String 获取错误码对应的消息
func (e ErrorCode) String() string {
	if msg, ok := errorCodeMessages[e]; ok {
		return msg
	}
	return "未知错误"
}

// GetCode 获取错误码数值
func (e ErrorCode) GetCode() int {
	return int(e)
}

// RespEntity 统一响应实体
type RespEntity struct {
	Code int         `json:"code"` // 0=成功, 其他=错误
	Data interface{} `json:"data"` // 响应数据
	Msg  string      `json:"msg"`  // 响应消息
}

// Result 通用响应
func Result(code ErrorCode, data interface{}, msg string, c *gin.Context) {
	if msg == "" {
		msg = code.String()
	}
	c.JSON(http.StatusOK, RespEntity{
		Code: int(code),
		Data: data,
		Msg:  msg,
	})
}

// Ok 成功响应 (无数据)
func Ok(c *gin.Context) {
	Result(Success, map[string]interface{}{}, "操作成功", c)
}

// OkWithMessage 成功响应 (带消息)
func OkWithMessage(message string, c *gin.Context) {
	Result(Success, map[string]interface{}{}, message, c)
}

// OkWithData 成功响应 (带数据)
func OkWithData(data interface{}, c *gin.Context) {
	Result(Success, data, "操作成功", c)
}

// OkWithDetailed 成功响应 (带数据和消息)
func OkWithDetailed(data interface{}, message string, c *gin.Context) {
	Result(Success, data, message, c)
}

// Error 错误响应
func Error(err error, c *gin.Context) {
	if err == nil {
		Ok(c)
		return
	}

	// 检查是否是自定义错误
	if agamottoErr, ok := err.(AgamottoError); ok {
		c.JSON(http.StatusOK, RespEntity{
			Code: agamottoErr.GetCode(),
			Data: nil,
			Msg:  agamottoErr.Error(),
		})
		return
	}

	// 默认系统错误
	Result(SystemError, gin.H{}, err.Error(), c)
}

// ErrorWithCode 错误响应 (指定错误码)
func ErrorWithCode(code ErrorCode, err error, c *gin.Context) {
	if err == nil {
		Ok(c)
		return
	}
	Result(code, gin.H{}, err.Error(), c)
}

// FailWithMessage 失败响应 (系统错误)
func FailWithMessage(message string, c *gin.Context) {
	Result(SystemError, map[string]interface{}{}, message, c)
}

// FailWithDetailed 失败响应 (带数据)
func FailWithDetailed(data interface{}, message string, c *gin.Context) {
	Result(SystemError, data, message, c)
}

// ResultCodeMessage 根据错误码返回响应
func ResultCodeMessage(code ErrorCode, message string, c *gin.Context) {
	if code == Success {
		OkWithData(map[string]interface{}{}, c)
	} else {
		Result(code, map[string]interface{}{}, message, c)
	}
}

// NotFound 404响应
func NotFound(c *gin.Context) {
	Result(NotFoundError, gin.H{}, "请求的资源不存在", c)
}

// Unauthorized 401响应
func UnauthorizedResponse(c *gin.Context, message string) {
	if message == "" {
		message = "未授权，请先登录"
	}
	Result(Unauthorized, gin.H{}, message, c)
}

// Forbidden 403响应
func ForbiddenResponse(c *gin.Context, message string) {
	if message == "" {
		message = "禁止访问"
	}
	Result(Forbidden, gin.H{}, message, c)
}

// RateLimit 429响应
func RateLimit(c *gin.Context, message string) {
	if message == "" {
		message = "请求过于频繁，请稍后重试"
	}
	Result(RateLimited, gin.H{}, message, c)
}

// AgamottoError 自定义错误接口
type AgamottoError interface {
	Error() string
	GetCode() int
}

// RespError 响应错误
type RespError struct {
	Code    ErrorCode
	Message string
}

func (e RespError) Error() string {
	return e.Message
}

func (e RespError) GetCode() int {
	return int(e.Code)
}

// NewError 创建自定义错误
func NewError(code ErrorCode, message string) error {
	return RespError{
		Code:    code,
		Message: message,
	}
}

// WrapError 包装错误
func WrapError(code ErrorCode, err error) error {
	if err == nil {
		return nil
	}
	return RespError{
		Code:    code,
		Message: err.Error(),
	}
}

// ResponseHandler 响应处理器 (用于中间件)
type ResponseHandler struct {
	gin.ResponseWriter
	StatusCode int
	Body       []byte
}

// Write 重写Write方法
func (r *ResponseHandler) Write(b []byte) (int, error) {
	r.Body = append(r.Body, b...)
	return r.ResponseWriter.Write(b)
}

// WriteHeader 重写WriteHeader方法
func (r *ResponseHandler) WriteHeader(code int) {
	r.StatusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// LogResponse 记录响应日志
func LogResponse(c *gin.Context) {
	start := c.GetTime("start")
	latency := c.GetDuration("latency")
	status := c.Writer.Status()

	log.Info().
		Str("method", c.Request.Method).
		Str("path", c.Request.URL.Path).
		Int("status", status).
		Dur("latency", latency).
		Str("ip", c.ClientIP()).
		Str("start", start.String()).
		Msg("请求完成")
}
