// Package api 定义 SSPanel 适配器与控制器之间使用的接口和通用模型。

package api

// API 定义 SSPanel 客户端需要提供的能力。
type API interface {
	// GetNodeInfo 拉取节点/用户/规则信息
	GetNodeInfo() (nodeInfo *NodeInfo, err error)
	GetUserList() (userList *[]UserInfo, err error)
	// ReportNodeStatus 上报节点状态、在线用户、流量与违规
	ReportNodeStatus(nodeStatus *NodeStatus) (err error)
	ReportNodeOnlineUsers(onlineUser *[]OnlineUser) (err error)
	ReportUserTraffic(userTraffic *[]UserTraffic) (err error)
	// Describe 面板描述信息
	Describe() ClientInfo
	// GetNodeRule 规则与违规上报
	GetNodeRule() (ruleList *[]DetectRule, err error)
	ReportIllegal(detectResultList *[]DetectResult) (err error)
	// Debug 调试开关
	Debug()
}
