// Package api 定义面板 API 接口与通用模型
// 新面板只需实现 API 接口即可接入。

package api

// API 定义不同面板所需的统一接口
type API interface {
	// GetNodeInfo 拉取节点/用户/规则信息
	GetNodeInfo() (nodeInfo *NodeInfo, err error)
	GetUserList() (userList *[]UserInfo, err error)
	// ReportNodeStatus 上报节点状态、在线用户、流量与违规
	ReportNodeStatus(nodeStatus *NodeStatus) (err error)
	ReportNodeOnlineUsers(onlineUser *[]OnlineUser) (err error)
	ReportUserTraffic(userTraffic *[]UserTraffic) (err error)
	// 面板描述信息
	Describe() ClientInfo
	// 规则与违规上报
	GetNodeRule() (ruleList *[]DetectRule, err error)
	ReportIllegal(detectResultList *[]DetectResult) (err error)
	// 调试开关
	Debug()
}
