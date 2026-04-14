package response

import (
	"errors"
	"net/http"

	"github.com/base-go/base/pkg/apperror"
	"github.com/gin-gonic/gin"
)

// Response là cấu trúc JSON response chuẩn cho toàn bộ hệ thống.
type Response struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Success trả về response thành công với data.
func Success(c *gin.Context, status int, data interface{}) {
	c.JSON(status, Response{
		Code:    "SUCCESS",
		Message: "success",
		Data:    data,
	})
}

// OK trả về 200 với data.
func OK(c *gin.Context, data interface{}) {
	Success(c, http.StatusOK, data)
}

// Accepted trả về 202 với data.
func Accepted(c *gin.Context, data interface{}) {
	Success(c, http.StatusAccepted, data)
}

// Created trả về 201 với data.
func Created(c *gin.Context, data interface{}) {
	Success(c, http.StatusCreated, data)
}

// NoContent trả về 204 không có body.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Error trả về response lỗi. Tự động map AppError sang HTTP status phù hợp.
func Error(c *gin.Context, err error) {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.HTTPStatus(), Response{
			Code:    string(appErr.Code),
			Message: appErr.Message,
		})
		return
	}

	// Lỗi không xác định → 500 Internal Server Error.
	// Không để lộ chi tiết lỗi internal ra ngoài.
	c.JSON(http.StatusInternalServerError, Response{
		Code:    string(apperror.CodeInternalError),
		Message: "an unexpected error occurred",
	})
}

// AbortWithError abort request và trả về lỗi (dùng trong middleware).
func AbortWithError(c *gin.Context, err error) {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		c.AbortWithStatusJSON(appErr.HTTPStatus(), Response{
			Code:    string(appErr.Code),
			Message: appErr.Message,
		})
		return
	}

	c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
		Code:    string(apperror.CodeInternalError),
		Message: "an unexpected error occurred",
	})
}
