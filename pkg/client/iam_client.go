package client

import (
	"context"

	"github.com/vera-byte/vgo-gateway/pkg/model"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// IAMClient IAM服务客户端接口
type IAMClient interface {
	// VerifyToken 验证访问令牌
	// 参数: ctx 上下文, token 访问令牌
	// 返回值: *model.User 用户信息, error 错误信息
	VerifyToken(ctx context.Context, token string) (*model.User, error)

	// Login 用户登录
	// 参数: ctx 上下文, username 用户名, password 密码
	// 返回值: *model.LoginResponse 登录响应, error 错误信息
	Login(ctx context.Context, username, password string) (*model.LoginResponse, error)

	// GetUserInfo 获取用户信息
	// 参数: ctx 上下文, userID 用户ID
	// 返回值: *model.User 用户信息, error 错误信息
	GetUserInfo(ctx context.Context, userID string) (*model.User, error)
}

// iamClient IAM客户端实现
type iamClient struct {
	conn   *grpc.ClientConn
	config IAMConfig
}

// IAMConfig IAM客户端配置
type IAMConfig struct {
	Endpoint string
	Timeout  int
}

// NewIAMClient 创建新的IAM客户端
// 参数: cfg IAM配置
// 返回值: IAMClient 客户端接口, error 错误信息
func NewIAMClient(cfg IAMConfig) (IAMClient, error) {
	// 创建非阻塞连接，允许服务在后台启动
	conn, err := grpc.Dial(cfg.Endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &iamClient{
		conn:   conn,
		config: cfg,
	}, nil
}

// VerifyToken 验证访问令牌
func (c *iamClient) VerifyToken(ctx context.Context, token string) (*model.User, error) {
	// TODO: 实现gRPC调用验证令牌
	// 这里需要根据vgo-iam的proto定义来实现
	return &model.User{
		ID:       "admin",
		Username: "admin",
		Email:    "admin@example.com",
		Roles:    []string{"admin"},
	}, nil
}

// Login 用户登录
func (c *iamClient) Login(ctx context.Context, username, password string) (*model.LoginResponse, error) {
	// TODO: 实现gRPC调用登录
	// 这里需要根据vgo-iam的proto定义来实现
	return &model.LoginResponse{
		Token: "mock-jwt-token",
		User: &model.User{
			ID:       "admin",
			Username: username,
			Email:    "admin@example.com",
			Roles:    []string{"admin"},
		},
		ExpiresIn: 3600,
	}, nil
}

// GetUserInfo 获取用户信息
func (c *iamClient) GetUserInfo(ctx context.Context, userID string) (*model.User, error) {
	// TODO: 实现gRPC调用获取用户信息
	// 这里需要根据vgo-iam的proto定义来实现
	return &model.User{
		ID:       userID,
		Username: "admin",
		Email:    "admin@example.com",
		Roles:    []string{"admin"},
	}, nil
}

// Close 关闭连接
func (c *iamClient) Close() error {
	return c.conn.Close()
}
