package route

import (
	"net/http"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/internal/service"
	"github.com/acepanel/panel/v3/pkg/types"
)

// ContainerRoutes 容器路由
func ContainerRoutes(i do.Injector) (Endpoints, error) {
	container := do.MustInvoke[*service.ContainerService](i)
	compose := do.MustInvoke[*service.ContainerComposeService](i)
	network := do.MustInvoke[*service.ContainerNetworkService](i)
	image := do.MustInvoke[*service.ContainerImageService](i)
	volume := do.MustInvoke[*service.ContainerVolumeService](i)

	return Endpoints{
		// 容器
		{Method: http.MethodGet, Path: "/api/container/container", Handler: container.List,
			Summary: "容器列表", Tags: []string{"容器"},
			Response: service.Envelope[service.Page[types.Container]]{}},
		{Method: http.MethodGet, Path: "/api/container/container/search", Handler: container.Search,
			Summary: "搜索容器", Tags: []string{"容器"},
			Response: service.Envelope[service.Page[types.Container]]{}},
		{Method: http.MethodGet, Path: "/api/container/container/{id}", Handler: container.Inspect,
			Summary: "容器详情", Tags: []string{"容器"},
			Request: request.ContainerID{}},
		{Method: http.MethodPost, Path: "/api/container/container", Handler: container.Create,
			Summary: "创建容器", Tags: []string{"容器"},
			Request: request.ContainerCreate{}},
		{Method: http.MethodPut, Path: "/api/container/container/{id}", Handler: container.Update,
			Summary: "更新容器", Tags: []string{"容器"},
			Request: request.ContainerCreate{}},
		{Method: http.MethodDelete, Path: "/api/container/container/{id}", Handler: container.Remove,
			Summary: "删除容器", Tags: []string{"容器"},
			Request: request.ContainerID{}},
		{Method: http.MethodPost, Path: "/api/container/container/{id}/start", Handler: container.Start,
			Summary: "启动容器", Tags: []string{"容器"},
			Request: request.ContainerID{}},
		{Method: http.MethodPost, Path: "/api/container/container/{id}/stop", Handler: container.Stop,
			Summary: "停止容器", Tags: []string{"容器"},
			Request: request.ContainerID{}},
		{Method: http.MethodPost, Path: "/api/container/container/{id}/restart", Handler: container.Restart,
			Summary: "重启容器", Tags: []string{"容器"},
			Request: request.ContainerID{}},
		{Method: http.MethodPost, Path: "/api/container/container/{id}/pause", Handler: container.Pause,
			Summary: "暂停容器", Tags: []string{"容器"},
			Request: request.ContainerID{}},
		{Method: http.MethodPost, Path: "/api/container/container/{id}/unpause", Handler: container.Unpause,
			Summary: "恢复容器", Tags: []string{"容器"},
			Request: request.ContainerID{}},
		{Method: http.MethodPost, Path: "/api/container/container/{id}/kill", Handler: container.Kill,
			Summary: "杀死容器", Tags: []string{"容器"},
			Request: request.ContainerID{}},
		{Method: http.MethodPost, Path: "/api/container/container/{id}/rename", Handler: container.Rename,
			Summary: "重命名容器", Tags: []string{"容器"},
			Request: request.ContainerRename{}},
		{Method: http.MethodPost, Path: "/api/container/container/prune", Handler: container.Prune,
			Summary: "清理容器", Tags: []string{"容器"}},

		// 编排
		{Method: http.MethodGet, Path: "/api/container/compose", Handler: compose.List,
			Summary: "编排列表", Tags: []string{"容器编排"},
			Response: service.Envelope[service.Page[types.ContainerCompose]]{}},
		{Method: http.MethodGet, Path: "/api/container/compose/{name}", Handler: compose.Get,
			Summary: "获取编排", Tags: []string{"容器编排"},
			Request: request.ContainerComposeGet{}},
		{Method: http.MethodPost, Path: "/api/container/compose", Handler: compose.Create,
			Summary: "创建编排", Tags: []string{"容器编排"},
			Request: request.ContainerComposeCreate{}},
		{Method: http.MethodPut, Path: "/api/container/compose/{name}", Handler: compose.Update,
			Summary: "更新编排", Tags: []string{"容器编排"},
			Request: request.ContainerComposeUpdate{}},
		{Method: http.MethodPost, Path: "/api/container/compose/{name}/up", Handler: compose.Up,
			Summary: "启动编排", Tags: []string{"容器编排"},
			Request: request.ContainerComposeUp{}},
		{Method: http.MethodPost, Path: "/api/container/compose/{name}/down", Handler: compose.Down,
			Summary: "停止编排", Tags: []string{"容器编排"},
			Request: request.ContainerComposeDown{}},
		{Method: http.MethodDelete, Path: "/api/container/compose/{name}", Handler: compose.Remove,
			Summary: "删除编排", Tags: []string{"容器编排"},
			Request: request.ContainerComposeRemove{}},

		// 网络
		{Method: http.MethodGet, Path: "/api/container/network", Handler: network.List,
			Summary: "网络列表", Tags: []string{"容器网络"},
			Response: service.Envelope[service.Page[types.ContainerNetwork]]{}},
		{Method: http.MethodPost, Path: "/api/container/network", Handler: network.Create,
			Summary: "创建网络", Tags: []string{"容器网络"},
			Request: request.ContainerNetworkCreate{}},
		{Method: http.MethodDelete, Path: "/api/container/network/{id}", Handler: network.Remove,
			Summary: "删除网络", Tags: []string{"容器网络"},
			Request: request.ContainerNetworkID{}},
		{Method: http.MethodPost, Path: "/api/container/network/prune", Handler: network.Prune,
			Summary: "清理网络", Tags: []string{"容器网络"}},

		// 镜像
		{Method: http.MethodGet, Path: "/api/container/image", Handler: image.List,
			Summary: "镜像列表", Tags: []string{"容器镜像"},
			Response: service.Envelope[service.Page[types.ContainerImage]]{}},
		{Method: http.MethodGet, Path: "/api/container/image/exist", Handler: image.Exist,
			Summary: "镜像是否存在", Tags: []string{"容器镜像"},
			Request: request.ContainerImagePull{}},
		{Method: http.MethodPost, Path: "/api/container/image", Handler: image.Pull,
			Summary: "拉取镜像", Tags: []string{"容器镜像"},
			Request: request.ContainerImagePull{}},
		{Method: http.MethodDelete, Path: "/api/container/image/{id}", Handler: image.Remove,
			Summary: "删除镜像", Tags: []string{"容器镜像"},
			Request: request.ContainerImageID{}},
		{Method: http.MethodPost, Path: "/api/container/image/prune", Handler: image.Prune,
			Summary: "清理镜像", Tags: []string{"容器镜像"}},

		// 存储卷
		{Method: http.MethodGet, Path: "/api/container/volume", Handler: volume.List,
			Summary: "存储卷列表", Tags: []string{"容器存储卷"},
			Response: service.Envelope[service.Page[types.ContainerVolume]]{}},
		{Method: http.MethodPost, Path: "/api/container/volume", Handler: volume.Create,
			Summary: "创建存储卷", Tags: []string{"容器存储卷"},
			Request: request.ContainerVolumeCreate{}},
		{Method: http.MethodDelete, Path: "/api/container/volume/{id}", Handler: volume.Remove,
			Summary: "删除存储卷", Tags: []string{"容器存储卷"},
			Request: request.ContainerVolumeID{}},
		{Method: http.MethodPost, Path: "/api/container/volume/prune", Handler: volume.Prune,
			Summary: "清理存储卷", Tags: []string{"容器存储卷"}},
	}, nil
}
