package middleware

import (
	"github.com/gin-gonic/gin"
)

// DeprecationWarning 返回一个中间件，用于标记API为废弃状态
// migrateToPath: 新API的路径，用于引导用户迁移
func DeprecationWarning(migrateToPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 标记API为废弃状态
		c.Header("X-API-Deprecated", "true")

		// 设置废弃日期（RFC 8594格式）
		// 2026年6月1日后将正式下线
		c.Header("Deprecation", `date="2026-06-01"`)

		// 提供替代API的链接（RFC 8288格式）
		if migrateToPath != "" {
			c.Header("Link", `<`+migrateToPath+`>; rel="alternate"`)
		}

		// 添加Sunset头，表示API将在何时停止服务
		c.Header("Sunset", "Fri, 01 Jun 2026 00:00:00 GMT")

		c.Next()
	}
}
