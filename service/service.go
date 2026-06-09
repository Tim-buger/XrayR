// Package service 定义 XrayR 内部服务的统一生命周期接口。
package service

// Service 表示由 Panel 管理的可启动、可关闭服务。
type Service interface {
	Start() error
	Close() error
	Restart
}

// Restart 描述服务重启所需的启动和关闭能力。
type Restart interface {
	Start() error
	Close() error
}
