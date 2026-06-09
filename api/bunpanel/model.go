package bunpanel

import "encoding/json"

type Server struct {
	// 节点配置
	Port int `json:"serverPort"`
	Network string `json:"network"`
	Method string `json:"method"`
	Security string `json:"security"`
	Flow string 	`json:"flow"`
	WsSettings json.RawMessage `json:"wsSettings"`
	RealitySettings json.RawMessage `json:"realitySettings"`
	GrpcSettings json.RawMessage `json:"grpcSettings"`
	TcpSettings json.RawMessage `json:"tcpSettings"`
}

type WsSettings struct {
	// WebSocket 配置
	Path string `json:"path"`
	Headers struct {
		Host string `json:"Host"`
	} `json:"headers"`
}

type GrpcSettigns struct {
	// gRPC 配置
	ServiceName string `json:"serviceName"`
}

type TcpSettings struct {
	// TCP 头部配置
	Header json.RawMessage `json:"header"`
}

type RealitySettings struct {
	// REALITY 配置
	Show    bool            `json:"show"`
	Dest    string          `json:"dest"`
	Xver 	uint64 			`json:"xver"`
	ServerNames []string    `json:"serverNames"`
	PrivateKey string       `json:"privateKey"`
	MinClientVer string     `json:"minClientVer"`
	MaxClientVer string     `json:"maxClientVer"`
	MaxTimeDiff uint64     	`json:"maxTimeDiff"`
	ProxyProtocolVer uint64 `json:"proxyProtocolVer"`	
	ShortIds []string       `json:"shortIds"`
}

type User struct {
	// 用户下发信息
	ID int `json:"id"`
	UUID string `json:"uuid"`
	SpeedLimit float64 `json:"speedLimit"`
	DeviceLimit int `json:"ipLimit"`
	AliveIP int `json:"onlineIp"`
}

type OnlineUser struct {
	// 在线用户信息
	UID int    `json:"userId"`
	IP  string `json:"ip"`
}

// UserTraffic is the data structure of traffic
type UserTraffic struct {
	UID      int   `json:"userId"`
	Upload   int64 `json:"u"`
	Download int64 `json:"d"`
}

type Response struct {
	// 通用响应结构
	StatusCode int `json:"statusCode"`
	Datas json.RawMessage `json:"datas"`
}

type PostData struct {
	// 通用提交结构
	Data interface{} `json:"data"`
}
