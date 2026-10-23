package api

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, srv HTTPServer) {
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodPost,
				Path:    "/upload",
				Handler: UploadDocumentHandler(srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/page/query",
				Handler: QueryDocumentPageHandler(srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/detail/query",
				Handler: QueryDocumentDetailHandler(srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/delete",
				Handler: DeleteDocumentHandler(srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/strategy/plan/query",
				Handler: QueryStrategyPlanHandler(srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/strategy/confirm",
				Handler: ConfirmStrategyHandler(srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/index/build",
				Handler: BuildIndexHandler(srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/chunk/query",
				Handler: QueryDocumentChunksHandler(srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/chunk/detail/query",
				Handler: QueryDocumentChunkDetailHandler(srv),
			},
			{
				Method:  http.MethodGet,
				Path:    "/options",
				Handler: GetDocumentOptionsHandler(srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/task/log/query",
				Handler: QueryTaskLogsHandler(srv),
			},
		},
		rest.WithPrefix("/manage/document"),
	)
}
