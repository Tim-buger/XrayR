package newV2board

import (
	"encoding/json"
)

type serverConfig struct {
	// 节点下发的配置聚合
	shadowsocks
	v2ray
	trojan

	ServerPort int `json:"server_port"`
	BaseConfig struct {
		PushInterval int `json:"push_interval"`
		PullInterval int `json:"pull_interval"`
	} `json:"base_config"`
	Routes []route `json:"routes"`
}

type shadowsocks struct {
	// shadowsocks 配置
	Cipher       string `json:"cipher"`
	Obfs         string `json:"obfs"`
	ObfsSettings struct {
		Path string `json:"path"`
		Host string `json:"host"`
	} `json:"obfs_settings"`
	ServerKey string `json:"server_key"`
}

type v2ray struct {
	// v2ray/vless 传输配置
	Network         string `json:"network"`
	NetworkSettings struct {
		Path        string           `json:"path"`
		Host        string           `json:"host"`
		Headers     *json.RawMessage `json:"headers"`
		ServiceName string           `json:"serviceName"`
		Header      *json.RawMessage `json:"header"`
	} `json:"networkSettings"`
	VlessNetworkSettings struct {
		Path        string           `json:"path"`
		Host        string           `json:"host"`
		Headers     *json.RawMessage `json:"headers"`
		ServiceName string           `json:"serviceName"`
		Header      *json.RawMessage `json:"header"`
	} `json:"network_settings"`
	VlessFlow        string `json:"flow"`
	VlessTlsSettings struct {
		ServerPort string `json:"server_port"`
		Dest       string `json:"dest"`
		xVer       uint64 `json:"xver"`
		Sni        string `json:"server_name"`
		PrivateKey string `json:"private_key"`
		ShortId    string `json:"short_id"`
	} `json:"tls_settings"`
	Tls int `json:"tls"`
}

type trojan struct {
	// trojan 配置
	Host       string `json:"host"`
	ServerName string `json:"server_name"`
}

type route struct {
	// 路由规则
	Id          int      `json:"id"`
	Match       []string `json:"match"`
	Action      string   `json:"action"`
	ActionValue string   `json:"action_value"`
}

type user struct {
	// 用户下发信息
	Id         int    `json:"id"`
	Uuid       string `json:"uuid"`
	SpeedLimit int    `json:"speed_limit"`
}
