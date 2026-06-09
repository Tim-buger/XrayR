package pmpanel

import "encoding/json"

// NodeInfoResponse 节点信息响应
type NodeInfoResponse struct {
	Class           int     `json:"clazz"`
	SpeedLimit      float64 `json:"speedlimit"`
	Method          string  `json:"method"`
	TrafficRate     float64 `json:"trafficRate"`
	RawServerString string  `json:"outServer"`
	Port            uint32  `json:"outPort"`
	AlterId         uint16  `json:"alterId"`
	Network         string  `json:"network"`
	Security        string  `json:"security"`
	Host            string  `json:"host"`
	Path            string  `json:"path"`
	Grpc            bool    `json:"grpc"`
	Sni             string  `json:"sni"`
}

// UserResponse 用户信息响应
type UserResponse struct {
	ID          int     `json:"id"`
	Passwd      string  `json:"passwd"`
	SpeedLimit  float64 `json:"nodeSpeedlimit"`
	DeviceLimit int     `json:"nodeConnector"`
}

// Response 通用响应结构
type Response struct {
	Ret  uint            `json:"ret"`
	Data json.RawMessage `json:"data"`
}

// PostData 上报数据结构
type PostData struct {
	Type    string      `json:"type"`
	NodeId  int         `json:"nodeId"`
	Users   interface{} `json:"users"`
	Onlines interface{} `json:"onlines"`
}

// SystemLoad 系统负载信息
type SystemLoad struct {
	Uptime string `json:"uptime"`
	Load   string `json:"load"`
}

// OnlineUser 在线用户信息
type OnlineUser struct {
	UID int    `json:"user_id"`
	IP  string `json:"ip"`
}

// UserTraffic 用户流量上报结构
type UserTraffic struct {
	UID      int    `json:"id"`
	Upload   int64  `json:"up"`
	Download int64  `json:"down"`
	Ip       string `json:"ip"`
}

type RuleItem struct {
	// 规则项
	ID      int    `json:"id"`
	Content string `json:"regex"`
}

type IllegalItem struct {
	// 违规上报项
	ID  int `json:"list_id"`
	UID int `json:"user_id"`
}
