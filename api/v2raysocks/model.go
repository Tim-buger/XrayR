package v2raysocks

type UserTraffic struct {
	// 用户流量上报
	UID      int   `json:"uid"`
	Upload   int64 `json:"u"`
	Download int64 `json:"d"`
}

type NodeStatus struct {
	// 节点状态
	CPU    string `json:"cpu"`
	Mem    string `json:"mem"`
	Net    string `json:"net"`
	Disk   string `json:"disk"`
	Uptime int    `json:"uptime"`
}

type NodeOnline struct {
	// 在线用户
	UID int    `json:"uid"`
	IP  string `json:"ip"`
}

type IllegalItem struct {
	// 违规上报项
	UID int `json:"uid"`
}

type REALITYConfig struct {
	// REALITY 配置
	Dest             string   `json:"dest,omitempty"`
	ProxyProtocolVer uint64   `json:"proxy_protocol_ver,omitempty"`
	ServerNames      []string `json:"server_names,omitempty"`
	PrivateKey       string   `json:"private_key,omitempty"`
	MinClientVer     string   `json:"min_client_ver,omitempty"`
	MaxClientVer     string   `json:"max_client_ver,omitempty"`
	MaxTimeDiff      uint64   `json:"max_time_diff,omitempty"`
	ShortIds         []string `json:"short_ids,omitempty"`
}
