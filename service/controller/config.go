package controller

import (
	"github.com/XrayR-project/XrayR/common/limiter"
	"github.com/XrayR-project/XrayR/common/mylego"
)

type Config struct {
	// 监听与发送相关配置
	ListenIP                  string                           `mapstructure:"ListenIP"`
	SendIP                    string                           `mapstructure:"SendIP"`
	// 定时更新间隔（秒）
	UpdatePeriodic            int                              `mapstructure:"UpdatePeriodic"`
	// 证书管理（acme/自签/文件）
	CertConfig                *mylego.CertConfig               `mapstructure:"CertConfig"`
	// DNS 解析策略
	EnableDNS                 bool                             `mapstructure:"EnableDNS"`
	DNSType                   string                           `mapstructure:"DNSType"`
	// 行为开关
	DisableUploadTraffic      bool                             `mapstructure:"DisableUploadTraffic"`
	DisableGetRule            bool                             `mapstructure:"DisableGetRule"`
	EnableProxyProtocol       bool                             `mapstructure:"EnableProxyProtocol"`
	EnableFallback            bool                             `mapstructure:"EnableFallback"`
	DisableIVCheck            bool                             `mapstructure:"DisableIVCheck"`
	DisableSniffing           bool                             `mapstructure:"DisableSniffing"`
	// 自动限速
	AutoSpeedLimitConfig      *AutoSpeedLimitConfig            `mapstructure:"AutoSpeedLimitConfig"`
	// 设备数限制（支持全局限制）
	GlobalDeviceLimitConfig   *limiter.GlobalDeviceLimitConfig `mapstructure:"GlobalDeviceLimitConfig"`
	// VLESS/Trojan fallback
	FallBackConfigs           []*FallBackConfig                `mapstructure:"FallBackConfigs"`
	// REALITY 开关与配置
	DisableLocalREALITYConfig bool                             `mapstructure:"DisableLocalREALITYConfig"`
	EnableREALITY             bool                             `mapstructure:"EnableREALITY"`
	REALITYConfigs            *REALITYConfig                   `mapstructure:"REALITYConfigs"`
}

type AutoSpeedLimitConfig struct {
	// 用户超速判定与限速策略
	Limit         int `mapstructure:"Limit"` // mbps
	WarnTimes     int `mapstructure:"WarnTimes"`
	LimitSpeed    int `mapstructure:"LimitSpeed"`    // mbps
	LimitDuration int `mapstructure:"LimitDuration"` // minute
}

type FallBackConfig struct {
	// Fallback 匹配条件与转发目标
	SNI              string `mapstructure:"SNI"`
	Alpn             string `mapstructure:"Alpn"`
	Path             string `mapstructure:"Path"`
	Dest             string `mapstructure:"Dest"`
	ProxyProtocolVer uint64 `mapstructure:"ProxyProtocolVer"`
}

type REALITYConfig struct {
	// REALITY 协议相关参数
	Show             bool     `mapstructure:"Show"`
	Dest             string   `mapstructure:"Dest"`
	ProxyProtocolVer uint64   `mapstructure:"ProxyProtocolVer"`
	ServerNames      []string `mapstructure:"ServerNames"`
	PrivateKey       string   `mapstructure:"PrivateKey"`
	MinClientVer     string   `mapstructure:"MinClientVer"`
	MaxClientVer     string   `mapstructure:"MaxClientVer"`
	MaxTimeDiff      uint64   `mapstructure:"MaxTimeDiff"`
	ShortIds         []string `mapstructure:"ShortIds"`
}
