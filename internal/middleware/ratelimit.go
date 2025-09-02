package middleware

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimiter 速率限制器接口
type RateLimiter interface {
	// Allow 检查是否允许请求
	Allow(ctx context.Context, key string) (bool, error)
	// AllowN 检查是否允许N个请求
	AllowN(ctx context.Context, key string, n int) (bool, error)
	// Reset 重置指定key的限制
	Reset(ctx context.Context, key string) error
	// GetRemaining 获取剩余请求数
	GetRemaining(ctx context.Context, key string) (int, error)
}

// RedisRateLimiter Redis实现的速率限制器
type RedisRateLimiter struct {
	client   *redis.Client
	limit    int           // 限制数量
	window   time.Duration // 时间窗口
	prefix   string        // key前缀
}

// NewRedisRateLimiter 创建Redis速率限制器
// 参数:
//   - client: Redis客户端
//   - limit: 限制数量
//   - window: 时间窗口
//   - prefix: key前缀
// 返回值:
//   - *RedisRateLimiter: Redis速率限制器实例
func NewRedisRateLimiter(client *redis.Client, limit int, window time.Duration, prefix string) *RedisRateLimiter {
	return &RedisRateLimiter{
		client: client,
		limit:  limit,
		window: window,
		prefix: prefix,
	}
}

// Allow 检查是否允许请求
// 参数:
//   - ctx: 上下文
//   - key: 限流key
// 返回值:
//   - bool: 是否允许
//   - error: 错误信息
func (r *RedisRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	return r.AllowN(ctx, key, 1)
}

// AllowN 检查是否允许N个请求
// 参数:
//   - ctx: 上下文
//   - key: 限流key
//   - n: 请求数量
// 返回值:
//   - bool: 是否允许
//   - error: 错误信息
func (r *RedisRateLimiter) AllowN(ctx context.Context, key string, n int) (bool, error) {
	fullKey := r.getKey(key)
	now := time.Now().Unix()
	windowStart := now - int64(r.window.Seconds())

	// 使用Lua脚本确保原子性
	luaScript := `
		local key = KEYS[1]
		local window_start = tonumber(ARGV[1])
		local now = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])
		local count = tonumber(ARGV[4])
		
		-- 清理过期的记录
		redis.call('ZREMRANGEBYSCORE', key, 0, window_start)
		
		-- 获取当前窗口内的请求数
		local current = redis.call('ZCARD', key)
		
		-- 检查是否超过限制
		if current + count > limit then
			return {0, current}
		end
		
		-- 添加新的请求记录
		for i = 1, count do
			redis.call('ZADD', key, now, now .. ':' .. i)
		end
		
		-- 设置过期时间
		redis.call('EXPIRE', key, math.ceil(ARGV[5]))
		
		return {1, current + count}
	`

	result, err := r.client.Eval(ctx, luaScript, []string{fullKey}, windowStart, now, r.limit, n, int(r.window.Seconds())).Result()
	if err != nil {
		return false, err
	}

	results := result.([]interface{})
	allowed := results[0].(int64) == 1

	return allowed, nil
}

// Reset 重置指定key的限制
// 参数:
//   - ctx: 上下文
//   - key: 限流key
// 返回值:
//   - error: 错误信息
func (r *RedisRateLimiter) Reset(ctx context.Context, key string) error {
	return r.client.Del(ctx, r.getKey(key)).Err()
}

// GetRemaining 获取剩余请求数
// 参数:
//   - ctx: 上下文
//   - key: 限流key
// 返回值:
//   - int: 剩余请求数
//   - error: 错误信息
func (r *RedisRateLimiter) GetRemaining(ctx context.Context, key string) (int, error) {
	fullKey := r.getKey(key)
	now := time.Now().Unix()
	windowStart := now - int64(r.window.Seconds())

	// 清理过期记录并获取当前计数
	luaScript := `
		local key = KEYS[1]
		local window_start = tonumber(ARGV[1])
		local limit = tonumber(ARGV[2])
		
		-- 清理过期的记录
		redis.call('ZREMRANGEBYSCORE', key, 0, window_start)
		
		-- 获取当前窗口内的请求数
		local current = redis.call('ZCARD', key)
		
		return limit - current
	`

	result, err := r.client.Eval(ctx, luaScript, []string{fullKey}, windowStart, r.limit).Result()
	if err != nil {
		return 0, err
	}

	remaining := int(result.(int64))
	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}

// getKey 获取完整的key
// 参数:
//   - key: 原始key
// 返回值:
//   - string: 完整的key
func (r *RedisRateLimiter) getKey(key string) string {
	return fmt.Sprintf("%s:%s", r.prefix, key)
}

// MemoryRateLimiter 内存实现的速率限制器
type MemoryRateLimiter struct {
	limit    int
	window   time.Duration
	requests map[string][]time.Time
}

// NewMemoryRateLimiter 创建内存速率限制器
// 参数:
//   - limit: 限制数量
//   - window: 时间窗口
// 返回值:
//   - *MemoryRateLimiter: 内存速率限制器实例
func NewMemoryRateLimiter(limit int, window time.Duration) *MemoryRateLimiter {
	return &MemoryRateLimiter{
		limit:    limit,
		window:   window,
		requests: make(map[string][]time.Time),
	}
}

// Allow 检查是否允许请求
// 参数:
//   - ctx: 上下文
//   - key: 限流key
// 返回值:
//   - bool: 是否允许
//   - error: 错误信息
func (m *MemoryRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	return m.AllowN(ctx, key, 1)
}

// AllowN 检查是否允许N个请求
// 参数:
//   - ctx: 上下文
//   - key: 限流key
//   - n: 请求数量
// 返回值:
//   - bool: 是否允许
//   - error: 错误信息
func (m *MemoryRateLimiter) AllowN(ctx context.Context, key string, n int) (bool, error) {
	now := time.Now()
	windowStart := now.Add(-m.window)

	// 清理过期请求
	if requests, exists := m.requests[key]; exists {
		validRequests := make([]time.Time, 0)
		for _, req := range requests {
			if req.After(windowStart) {
				validRequests = append(validRequests, req)
			}
		}
		m.requests[key] = validRequests
	}

	// 检查是否超过限制
	currentCount := len(m.requests[key])
	if currentCount+n > m.limit {
		return false, nil
	}

	// 添加新请求
	for i := 0; i < n; i++ {
		m.requests[key] = append(m.requests[key], now)
	}

	return true, nil
}

// Reset 重置指定key的限制
// 参数:
//   - ctx: 上下文
//   - key: 限流key
// 返回值:
//   - error: 错误信息
func (m *MemoryRateLimiter) Reset(ctx context.Context, key string) error {
	delete(m.requests, key)
	return nil
}

// GetRemaining 获取剩余请求数
// 参数:
//   - ctx: 上下文
//   - key: 限流key
// 返回值:
//   - int: 剩余请求数
//   - error: 错误信息
func (m *MemoryRateLimiter) GetRemaining(ctx context.Context, key string) (int, error) {
	now := time.Now()
	windowStart := now.Add(-m.window)

	// 清理过期请求
	if requests, exists := m.requests[key]; exists {
		validRequests := make([]time.Time, 0)
		for _, req := range requests {
			if req.After(windowStart) {
				validRequests = append(validRequests, req)
			}
		}
		m.requests[key] = validRequests
		return m.limit - len(validRequests), nil
	}

	return m.limit, nil
}

// RateLimitConfig 速率限制配置
type RateLimitConfig struct {
	Enabled   bool          `yaml:"enabled" json:"enabled"`
	Type      string        `yaml:"type" json:"type"` // "redis" or "memory"
	Limit     int           `yaml:"limit" json:"limit"`
	Window    time.Duration `yaml:"window" json:"window"`
	Prefix    string        `yaml:"prefix" json:"prefix"`
	RedisAddr string        `yaml:"redis_addr" json:"redis_addr"`
	RedisDB   int           `yaml:"redis_db" json:"redis_db"`
	RedisPass string        `yaml:"redis_pass" json:"redis_pass"`
}

// DefaultRateLimitConfig 默认速率限制配置
// 返回值:
//   - *RateLimitConfig: 默认配置
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Enabled: false,
		Type:    "memory",
		Limit:   100,
		Window:  time.Minute,
		Prefix:  "ratelimit",
	}
}

// NewRateLimiter 创建新的限流器
// 参数: config RateLimitConfig 限流配置
// 返回值: RateLimiter 限流器接口, error 错误信息
func NewRateLimiter(config RateLimitConfig) (RateLimiter, error) {
	if !config.Enabled {
		return &NoOpRateLimiter{}, nil
	}

	switch config.Type {
	case "redis":
		client := redis.NewClient(&redis.Options{
			Addr:     config.RedisAddr,
			DB:       config.RedisDB,
			Password: config.RedisPass,
		})
		return NewRedisRateLimiter(client, config.Limit, config.Window, config.Prefix), nil
	case "memory":
		return NewMemoryRateLimiter(config.Limit, config.Window), nil
	default:
		return nil, fmt.Errorf("unsupported rate limiter type: %s", config.Type)
	}
}

// NoOpRateLimiter 无操作速率限制器（用于禁用限流时）
type NoOpRateLimiter struct{}

// Allow 总是允许请求
func (noop *NoOpRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	return true, nil
}

// AllowN 总是允许N个请求
func (noop *NoOpRateLimiter) AllowN(ctx context.Context, key string, count int) (bool, error) {
	return true, nil
}

// Reset 重置（无操作）
func (noop *NoOpRateLimiter) Reset(ctx context.Context, key string) error {
	return nil
}

// GetRemaining 获取剩余请求数（总是返回最大值）
func (noop *NoOpRateLimiter) GetRemaining(ctx context.Context, key string) (int, error) {
	return 1000000, nil
}

// KeyFunc 生成限流key的函数类型
type KeyFunc func(c *gin.Context) string

// DefaultKeyFunc 默认的key生成函数（基于IP地址）
// 参数:
//   - c: Gin上下文
// 返回值:
//   - string: 限流key
func DefaultKeyFunc(c *gin.Context) string {
	// 尝试获取真实IP
	if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
		// 取第一个IP（原始客户端IP）
		ips := strings.Split(ip, ",")
		if len(ips) > 0 {
			return fmt.Sprintf("ip:%s", strings.TrimSpace(ips[0]))
		}
	}
	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		return fmt.Sprintf("ip:%s", ip)
	}

	// 使用RemoteAddr
	if ip, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil {
		return fmt.Sprintf("ip:%s", ip)
	}

	return fmt.Sprintf("ip:%s", c.ClientIP())
}

// UserKeyFunc 基于用户ID的key生成函数
// 参数:
//   - c: Gin上下文
// 返回值:
//   - string: 限流key
func UserKeyFunc(c *gin.Context) string {
	// 尝试从JWT token或session中获取用户ID
	if userID := c.GetString("user_id"); userID != "" {
		return fmt.Sprintf("user:%s", userID)
	}
	// 如果没有用户ID，回退到IP
	return DefaultKeyFunc(c)
}

// PathKeyFunc 基于请求路径的key生成函数
// 参数:
//   - c: Gin上下文
// 返回值:
//   - string: 限流key
func PathKeyFunc(c *gin.Context) string {
	return fmt.Sprintf("path:%s:%s", c.Request.Method, c.Request.URL.Path)
}

// CombinedKeyFunc 组合多个key生成函数
// 参数:
//   - funcs: key生成函数列表
// 返回值:
//   - KeyFunc: 组合后的key生成函数
func CombinedKeyFunc(funcs ...KeyFunc) KeyFunc {
	return func(c *gin.Context) string {
		keys := make([]string, len(funcs))
		for i, fn := range funcs {
			keys[i] = fn(c)
		}
		return strings.Join(keys, ":")
	}
}

// RateLimitMiddleware 速率限制中间件
// 参数:
//   - limiter: 速率限制器
//   - keyFunc: key生成函数
// 返回值:
//   - gin.HandlerFunc: Gin中间件函数
func RateLimitMiddleware(limiter RateLimiter, keyFunc KeyFunc) gin.HandlerFunc {
	if keyFunc == nil {
		keyFunc = DefaultKeyFunc
	}

	return func(c *gin.Context) {
		key := keyFunc(c)
		allowed, err := limiter.Allow(c.Request.Context(), key)
		if err != nil {
			c.JSON(500, gin.H{"error": "Rate limiter error"})
			c.Abort()
			return
		}

		if !allowed {
			// 获取剩余请求数用于响应头
			remaining, _ := limiter.GetRemaining(c.Request.Context(), key)
			c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
			c.JSON(429, gin.H{
				"error":   "Too Many Requests",
				"message": "Rate limit exceeded",
			})
			c.Abort()
			return
		}

		// 添加速率限制相关的响应头
		remaining, _ := limiter.GetRemaining(c.Request.Context(), key)
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))

		c.Next()
	}
}