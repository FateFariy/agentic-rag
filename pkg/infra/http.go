package infra

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest"
)

// NewRestServer 创建带统一 CORS 配置的 go-zero rest.Server，但不注册任何路由。
// 各微服务调用对应 api 包的 RegisterHandlers 仅注册本服务路由。
func NewRestServer(c rest.RestConf) *rest.Server {
	opts := []rest.RunOption{
		rest.WithCorsHeaders("*"),
		rest.WithCustomCors(func(header http.Header) {
			header.Add("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token,Authorization,Token,X-Token,X-User-Id,OS,Platform, Version")
			header.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS,PATCH")
			header.Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")
		}, nil, "*"),
	}
	return rest.MustNewServer(c, opts...)
}
