package middleware

import (
	"net/http"
	"strings"

	"github.com/vera-byte/vgo-gateway/pkg/client"
	"github.com/vera-byte/vgo-gateway/pkg/model"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 认证中间件
// 参数: iamClient IAM客户端
// 返回值: gin.HandlerFunc 中间件函数
func AuthMiddleware(iamClient client.IAMClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取Authorization头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, model.ErrorResponse{
				Code:    http.StatusUnauthorized,
				Message: "Missing authorization header",
			})
			c.Abort()
			return
		}

		// 检查Bearer token格式
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, model.ErrorResponse{
				Code:    http.StatusUnauthorized,
				Message: "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		token := parts[1]

		// 验证token
		user, err := iamClient.VerifyToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, model.ErrorResponse{
				Code:    http.StatusUnauthorized,
				Message: "Invalid token",
				Error:   err.Error(),
			})
			c.Abort()
			return
		}

		// 将用户信息存储到上下文中
		c.Set("user", user)
		c.Next()
	}
}

// RequireRole 角色权限中间件
// 参数: roles 需要的角色列表
// 返回值: gin.HandlerFunc 中间件函数
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userInterface, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, model.ErrorResponse{
				Code:    http.StatusUnauthorized,
				Message: "User not authenticated",
			})
			c.Abort()
			return
		}

		user, ok := userInterface.(*model.User)
		if !ok {
			c.JSON(http.StatusInternalServerError, model.ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "Invalid user data",
			})
			c.Abort()
			return
		}

		// 检查用户是否有所需角色
		hasRole := false
		for _, requiredRole := range roles {
			for _, userRole := range user.Roles {
				if userRole == requiredRole {
					hasRole = true
					break
				}
			}
			if hasRole {
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, model.ErrorResponse{
				Code:    http.StatusForbidden,
				Message: "Insufficient permissions",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
