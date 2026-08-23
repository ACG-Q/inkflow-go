package handlers

import "github.com/gin-gonic/gin"

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type PageData struct {
	Items    interface{} `json:"items"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(200, Response{Code: 0, Message: "success", Data: data})
}

func Error(c *gin.Context, httpCode, bizCode int, msg string) {
	c.JSON(httpCode, Response{Code: bizCode, Message: msg, Data: nil})
}

func Page(c *gin.Context, items interface{}, total int64, page, pageSize int) {
	c.JSON(200, Response{Code: 0, Message: "success", Data: PageData{
		Items: items, Total: total, Page: page, PageSize: pageSize,
	}})
}
