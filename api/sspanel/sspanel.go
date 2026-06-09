package sspanel

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/go-resty/resty/v2"

	"github.com/XrayR-project/XrayR/api"
)

var (
	firstPortRe  = regexp.MustCompile(`(?m)port=(?P<outport>\d+)#?`) // 外部端口
	secondPortRe = regexp.MustCompile(`(?m)port=\d+#(\d+)`)          // 内部偏移端口
	hostRe       = regexp.MustCompile(`(?m)host=([\w.]+)\|?`)        // 回源主机
)

// APIClient SSPANEL 面板 API 客户端
type APIClient struct {
	client              *resty.Client
	APIHost             string
	NodeID              int
	Key                 string
	NodeType            string
	EnableVless         bool
	VlessFlow           string
	SpeedLimit          float64
	DeviceLimit         int
	DisableCustomConfig bool
	LocalRuleList       []api.DetectRule
	LastReportOnline    map[int]int
	access              sync.Mutex
	version             string
	eTags               map[string]string
}

// New 创建 SSPANEL API 客户端
func New(apiConfig *api.Config) *APIClient {
	client := resty.New()

	client.SetRetryCount(3)
	if apiConfig.Timeout > 0 {
		client.SetTimeout(time.Duration(apiConfig.Timeout) * time.Second)
	} else {
		client.SetTimeout(5 * time.Second)
	}
	client.OnError(func(req *resty.Request, err error) {
		var v *resty.ResponseError
		if errors.As(err, &v) {
			// ResponseError 同时保留最后一次响应和原始请求错误。
			log.Print(v.Err)
		}
	})

	client.SetBaseURL(apiConfig.APIHost)
	// SSPanel 的不同版本可能读取 key 或 muKey，因此同时携带两者。
	client.SetQueryParam("key", apiConfig.Key)
	client.SetQueryParam("muKey", apiConfig.Key)
	// 本地规则会与面板下发规则合并。
	localRuleList := readLocalRuleList(apiConfig.RuleListPath)

	return &APIClient{
		client:              client,
		NodeID:              apiConfig.NodeID,
		Key:                 apiConfig.Key,
		APIHost:             apiConfig.APIHost,
		NodeType:            apiConfig.NodeType,
		EnableVless:         apiConfig.EnableVless,
		VlessFlow:           apiConfig.VlessFlow,
		SpeedLimit:          apiConfig.SpeedLimit,
		DeviceLimit:         apiConfig.DeviceLimit,
		LocalRuleList:       localRuleList,
		DisableCustomConfig: apiConfig.DisableCustomConfig,
		LastReportOnline:    make(map[int]int),
		eTags:               make(map[string]string),
	}
}

// readLocalRuleList 读取本地规则文件
func readLocalRuleList(path string) (LocalRuleList []api.DetectRule) {
	LocalRuleList = make([]api.DetectRule, 0)
	if path != "" {
		// 打开本地规则文件，每行是一条正则表达式。
		file, err := os.Open(path)

		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				log.Printf("Error when closing file: %s", err)
			}
		}(file)
		// 文件不可读时记录错误并返回空规则列表。
		if err != nil {
			log.Printf("Error when opening file: %s", err)
			return LocalRuleList
		}

		fileScanner := bufio.NewScanner(file)

		// 逐行编译规则。
		for fileScanner.Scan() {
			LocalRuleList = append(LocalRuleList, api.DetectRule{
				ID:      -1,
				Pattern: regexp.MustCompile(fileScanner.Text()),
			})
		}
		// 返回扫描过程中遇到的第一个错误。
		if err := fileScanner.Err(); err != nil {
			log.Fatalf("Error while reading file: %s", err)
			return
		}
	}

	return LocalRuleList
}

// Describe 返回客户端描述信息
func (c *APIClient) Describe() api.ClientInfo {
	return api.ClientInfo{APIHost: c.APIHost, NodeID: c.NodeID, Key: c.Key, NodeType: c.NodeType}
}

// Debug 打开 HTTP 调试日志
func (c *APIClient) Debug() {
	c.client.SetDebug(true)
}

func (c *APIClient) assembleURL(path string) string {
	// 拼接完整 URL
	return c.APIHost + path
}

func (c *APIClient) parseResponse(res *resty.Response, path string, err error) (*Response, error) {
	// 统一解析面板返回
	if err != nil {
		return nil, fmt.Errorf("request %s failed: %s", c.assembleURL(path), err)
	}

	if res.StatusCode() > 400 {
		body := res.Body()
		return nil, fmt.Errorf("request %s failed: %s, %v", c.assembleURL(path), string(body), err)
	}
	response := res.Result().(*Response)

	if response.Ret != 1 {
		res, _ := json.Marshal(&response)
		return nil, fmt.Errorf("ret %s invalid", string(res))
	}
	return response, nil
}

// GetNodeInfo 从 SSPanel 拉取节点配置并转换为通用 NodeInfo。
func (c *APIClient) GetNodeInfo() (nodeInfo *api.NodeInfo, err error) {
	path := fmt.Sprintf("/mod_mu/nodes/%d/info", c.NodeID)
	res, err := c.client.R().
		SetResult(&Response{}).
		SetHeader("If-None-Match", c.eTags["node"]).
		ForceContentType("application/json").
		Get(path)
	// ETag 用于避免重复解析未变化的节点配置；304 表示沿用旧数据。
	if res.StatusCode() == 304 {
		return nil, errors.New(api.NodeNotModified)
	}

	if res.Header().Get("ETag") != "" && res.Header().Get("ETag") != c.eTags["node"] {
		c.eTags["node"] = res.Header().Get("ETag")
	}

	response, err := c.parseResponse(res, path, err)
	if err != nil {
		return nil, err
	}

	nodeInfoResponse := new(NodeInfoResponse)

	if err := json.Unmarshal(response.Data, nodeInfoResponse); err != nil {
		return nil, fmt.Errorf("unmarshal %s failed: %s", reflect.TypeOf(nodeInfoResponse), err)
	}

	// 关闭 custom_config 或面板版本早于 2021.11 时，按旧版 server 字符串解析。
	c.version = nodeInfoResponse.Version
	var isExpired bool
	if compareVersion(c.version, "2021.11") == -1 {
		isExpired = true
	}

	if c.DisableCustomConfig || isExpired {
		if isExpired {
			log.Print("The panel version is expired, it is recommended to update immediately")
		}

		switch c.NodeType {
		case "V2ray":
			nodeInfo, err = c.ParseV2rayNodeResponse(nodeInfoResponse)
		case "Trojan":
			nodeInfo, err = c.ParseTrojanNodeResponse(nodeInfoResponse)
		case "Shadowsocks":
			nodeInfo, err = c.ParseSSNodeResponse(nodeInfoResponse)
		case "Shadowsocks-Plugin":
			nodeInfo, err = c.ParseSSPluginNodeResponse(nodeInfoResponse)
		default:
			return nil, fmt.Errorf("unsupported Node type: %s", c.NodeType)
		}
	} else {
		nodeInfo, err = c.ParseSSPanelNodeInfo(nodeInfoResponse)
		if err != nil {
			res, _ := json.Marshal(nodeInfoResponse)
			return nil, fmt.Errorf("parse node info failed: %s, \nError: %s, \nPlease check the doc of custom_config for help: https://xrayr-project.github.io/XrayR-doc/dui-jie-sspanel/sspanel/sspanel_custom_config", string(res), err)
		}
	}

	if err != nil {
		res, _ := json.Marshal(nodeInfoResponse)
		return nil, fmt.Errorf("parse node info failed: %s, \nError: %s", string(res), err)
	}

	return nodeInfo, nil
}

// GetUserList 从 SSPanel 拉取当前节点可用用户。
func (c *APIClient) GetUserList() (UserList *[]api.UserInfo, err error) {
	path := "/mod_mu/users"
	res, err := c.client.R().
		SetQueryParam("node_id", strconv.Itoa(c.NodeID)).
		SetHeader("If-None-Match", c.eTags["users"]).
		SetResult(&Response{}).
		ForceContentType("application/json").
		Get(path)
	// 304 表示用户列表未变化。
	if res.StatusCode() == 304 {
		return nil, errors.New(api.UserNotModified)
	}

	if res.Header().Get("ETag") != "" && res.Header().Get("ETag") != c.eTags["users"] {
		c.eTags["users"] = res.Header().Get("ETag")
	}

	response, err := c.parseResponse(res, path, err)
	if err != nil {
		return nil, err
	}

	userListResponse := new([]UserResponse)

	if err := json.Unmarshal(response.Data, userListResponse); err != nil {
		return nil, fmt.Errorf("unmarshal %s failed: %s", reflect.TypeOf(userListResponse), err)
	}
	userList, err := c.ParseUserListResponse(userListResponse)
	if err != nil {
		res, _ := json.Marshal(userListResponse)
		return nil, fmt.Errorf("parse user list failed: %s", string(res))
	}
	return userList, nil
}

// ReportNodeStatus 向支持该接口的旧版 SSPanel 上报系统状态。
func (c *APIClient) ReportNodeStatus(nodeStatus *api.NodeStatus) (err error) {
	// 2023.2 及以上版本不再需要这里的旧式状态上报。
	if compareVersion(c.version, "2023.2") == -1 {
		path := fmt.Sprintf("/mod_mu/nodes/%d/info", c.NodeID)
		systemLoad := SystemLoad{
			Uptime: strconv.FormatUint(nodeStatus.Uptime, 10),
			Load:   fmt.Sprintf("%.2f %.2f %.2f", nodeStatus.CPU/100, nodeStatus.Mem/100, nodeStatus.Disk/100),
		}

		res, err := c.client.R().
			SetBody(systemLoad).
			SetResult(&Response{}).
			ForceContentType("application/json").
			Post(path)

		_, err = c.parseResponse(res, path, err)
		if err != nil {
			return err
		}
	}
	return nil
}

// ReportNodeOnlineUsers 上报在线用户 IP，并记录本节点上次上报的设备数。
func (c *APIClient) ReportNodeOnlineUsers(onlineUserList *[]api.OnlineUser) error {
	c.access.Lock()
	defer c.access.Unlock()

	reportOnline := make(map[int]int)
	data := make([]OnlineUser, len(*onlineUserList))
	for i, user := range *onlineUserList {
		data[i] = OnlineUser{UID: user.UID, IP: user.IP}
		reportOnline[user.UID]++
	}
	c.LastReportOnline = reportOnline

	postData := &PostData{Data: data}
	path := "/mod_mu/users/aliveip"
	res, err := c.client.R().
		SetQueryParam("node_id", strconv.Itoa(c.NodeID)).
		SetBody(postData).
		SetResult(&Response{}).
		ForceContentType("application/json").
		Post(path)

	_, err = c.parseResponse(res, path, err)
	if err != nil {
		return err
	}

	return nil
}

// ReportUserTraffic 上报用户在本统计周期内的上下行流量。
func (c *APIClient) ReportUserTraffic(userTraffic *[]api.UserTraffic) error {

	data := make([]UserTraffic, len(*userTraffic))
	for i, traffic := range *userTraffic {
		data[i] = UserTraffic{
			UID:      traffic.UID,
			Upload:   traffic.Upload,
			Download: traffic.Download}
	}
	postData := &PostData{Data: data}
	path := "/mod_mu/users/traffic"
	res, err := c.client.R().
		SetQueryParam("node_id", strconv.Itoa(c.NodeID)).
		SetBody(postData).
		SetResult(&Response{}).
		ForceContentType("application/json").
		Post(path)
	_, err = c.parseResponse(res, path, err)
	if err != nil {
		return err
	}

	return nil
}

// GetNodeRule 合并本地规则与 SSPanel 下发的审计规则。
func (c *APIClient) GetNodeRule() (*[]api.DetectRule, error) {
	ruleList := c.LocalRuleList
	path := "/mod_mu/func/detect_rules"
	res, err := c.client.R().
		SetResult(&Response{}).
		SetHeader("If-None-Match", c.eTags["rules"]).
		ForceContentType("application/json").
		Get(path)

	// 304 表示审计规则未变化。
	if res.StatusCode() == 304 {
		return nil, errors.New(api.RuleNotModified)
	}

	if res.Header().Get("ETag") != "" && res.Header().Get("ETag") != c.eTags["rules"] {
		c.eTags["rules"] = res.Header().Get("ETag")
	}

	response, err := c.parseResponse(res, path, err)
	if err != nil {
		return nil, err
	}

	ruleListResponse := new([]RuleItem)

	if err := json.Unmarshal(response.Data, ruleListResponse); err != nil {
		return nil, fmt.Errorf("unmarshal %s failed: %s", reflect.TypeOf(ruleListResponse), err)
	}

	for _, r := range *ruleListResponse {
		ruleList = append(ruleList, api.DetectRule{
			ID:      r.ID,
			Pattern: regexp.MustCompile(r.Content),
		})
	}
	return &ruleList, nil
}

// ReportIllegal 上报用户命中的审计规则。
func (c *APIClient) ReportIllegal(detectResultList *[]api.DetectResult) error {

	data := make([]IllegalItem, len(*detectResultList))
	for i, r := range *detectResultList {
		data[i] = IllegalItem{
			ID:  r.RuleID,
			UID: r.UID,
		}
	}
	postData := &PostData{Data: data}
	path := "/mod_mu/users/detectlog"
	res, err := c.client.R().
		SetQueryParam("node_id", strconv.Itoa(c.NodeID)).
		SetBody(postData).
		SetResult(&Response{}).
		ForceContentType("application/json").
		Post(path)
	_, err = c.parseResponse(res, path, err)
	if err != nil {
		return err
	}
	return nil
}

// ParseV2rayNodeResponse 解析旧版 SSPanel 的 V2ray server 字符串。
func (c *APIClient) ParseV2rayNodeResponse(nodeInfoResponse *NodeInfoResponse) (*api.NodeInfo, error) {
	var enableTLS bool
	var path, host, transportProtocol, serviceName, HeaderType string
	var header json.RawMessage
	var speedLimit uint64 = 0
	if nodeInfoResponse.RawServerString == "" {
		return nil, fmt.Errorf("no server info in response")
	}
	// nodeInfo.RawServerString = strings.ToLower(nodeInfo.RawServerString)
	serverConf := strings.Split(nodeInfoResponse.RawServerString, ";")

	parsedPort, err := strconv.ParseInt(serverConf[1], 10, 32)
	if err != nil {
		return nil, err
	}
	port := uint32(parsedPort)

	parsedAlterID, err := strconv.ParseInt(serverConf[2], 10, 16)
	if err != nil {
		return nil, err
	}
	alterID := uint16(parsedAlterID)

	// 兼容 server 字符串中 TLS 和传输协议字段的不同排列。
	for _, value := range serverConf[3:5] {
		switch value {
		case "tls":
			enableTLS = true
		default:
			if value != "" {
				transportProtocol = value
			}
		}
	}
	extraServerConf := strings.Split(serverConf[5], "|")
	serviceName = ""
	for _, item := range extraServerConf {
		conf := strings.Split(item, "=")
		key := conf[0]
		if key == "" {
			continue
		}
		value := conf[1]
		switch key {
		case "path":
			rawPath := strings.Join(conf[1:], "=") // In case of the path strings contains the "="
			path = rawPath
		case "host":
			host = value
		case "servicename":
			serviceName = value
		case "headerType":
			HeaderType = value
		}
	}
	if c.SpeedLimit > 0 {
		speedLimit = uint64((c.SpeedLimit * 1000000) / 8)
	} else {
		speedLimit = uint64((nodeInfoResponse.SpeedLimit * 1000000) / 8)
	}

	if HeaderType != "" {
		headers := map[string]string{"type": HeaderType}
		header, err = json.Marshal(headers)
	}

	if err != nil {
		return nil, fmt.Errorf("marshal Header Type %s into config failed: %s", header, err)
	}

	// 转换为控制器统一使用的节点模型。
	nodeInfo := &api.NodeInfo{
		NodeType:          c.NodeType,
		NodeID:            c.NodeID,
		Port:              port,
		SpeedLimit:        speedLimit,
		AlterID:           alterID,
		TransportProtocol: transportProtocol,
		EnableTLS:         enableTLS,
		Path:              path,
		Host:              host,
		EnableVless:       c.EnableVless,
		VlessFlow:         c.VlessFlow,
		ServiceName:       serviceName,
		Header:            header,
	}

	return nodeInfo, nil
}

// ParseSSNodeResponse 解析旧版 Shadowsocks 节点配置。
func (c *APIClient) ParseSSNodeResponse(nodeInfoResponse *NodeInfoResponse) (*api.NodeInfo, error) {
	var port uint32 = 0
	var speedLimit uint64 = 0
	var method string
	path := "/mod_mu/users"
	res, err := c.client.R().
		SetQueryParam("node_id", strconv.Itoa(c.NodeID)).
		SetResult(&Response{}).
		ForceContentType("application/json").
		Get(path)

	response, err := c.parseResponse(res, path, err)
	if err != nil {
		return nil, err
	}

	userListResponse := new([]UserResponse)

	if err := json.Unmarshal(response.Data, userListResponse); err != nil {
		return nil, fmt.Errorf("unmarshal %s failed: %s", reflect.TypeOf(userListResponse), err)
	}

	// 单端口多用户模式取用户列表中的第一个端口作为服务端口。
	if len(*userListResponse) != 0 {
		port = (*userListResponse)[0].Port
	}

	if c.SpeedLimit > 0 {
		speedLimit = uint64((c.SpeedLimit * 1000000) / 8)
	} else {
		speedLimit = uint64((nodeInfoResponse.SpeedLimit * 1000000) / 8)
	}
	// 转换为控制器统一使用的节点模型。
	nodeInfo := &api.NodeInfo{
		NodeType:          c.NodeType,
		NodeID:            c.NodeID,
		Port:              port,
		SpeedLimit:        speedLimit,
		TransportProtocol: "tcp",
		CypherMethod:      method,
	}

	return nodeInfo, nil
}

// ParseSSPluginNodeResponse 解析旧版 Shadowsocks-Plugin 节点配置。
func (c *APIClient) ParseSSPluginNodeResponse(nodeInfoResponse *NodeInfoResponse) (*api.NodeInfo, error) {
	var enableTLS bool
	var path, host, transportProtocol string
	var speedLimit uint64 = 0

	serverConf := strings.Split(nodeInfoResponse.RawServerString, ";")
	parsedPort, err := strconv.ParseInt(serverConf[1], 10, 32)
	if err != nil {
		return nil, err
	}
	port := uint32(parsedPort)
	port-- // 插件模式占用相邻两个端口：底层 SS 端口和上层传输端口。
	if port <= 0 {
		return nil, fmt.Errorf("Shadowsocks-Plugin listen port must bigger than 1")
	}
	// 兼容 TLS、WebSocket 和 obfs 配置。
	for _, value := range serverConf[3:5] {
		switch value {
		case "tls":
			enableTLS = true
		case "ws":
			transportProtocol = "ws"
		case "obfs":
			transportProtocol = "tcp"
		}
	}

	extraServerConf := strings.Split(serverConf[5], "|")
	for _, item := range extraServerConf {
		conf := strings.Split(item, "=")
		key := conf[0]
		if key == "" {
			continue
		}
		value := conf[1]
		switch key {
		case "path":
			rawPath := strings.Join(conf[1:], "=") // In case of the path strings contains the "="
			path = rawPath
		case "host":
			host = value
		}
	}
	if c.SpeedLimit > 0 {
		speedLimit = uint64((c.SpeedLimit * 1000000) / 8)
	} else {
		speedLimit = uint64((nodeInfoResponse.SpeedLimit * 1000000) / 8)
	}

	// 转换为控制器统一使用的节点模型。
	nodeInfo := &api.NodeInfo{
		NodeType:          c.NodeType,
		NodeID:            c.NodeID,
		Port:              port,
		SpeedLimit:        speedLimit,
		TransportProtocol: transportProtocol,
		EnableTLS:         enableTLS,
		Path:              path,
		Host:              host,
	}

	return nodeInfo, nil
}

// ParseTrojanNodeResponse 解析旧版 Trojan server 字符串。
func (c *APIClient) ParseTrojanNodeResponse(nodeInfoResponse *NodeInfoResponse) (*api.NodeInfo, error) {
	// 域名或IP;port=连接端口#偏移端口|host=xx
	// gz.aaa.com;port=443#12345|host=hk.aaa.com
	var p, host, outsidePort, insidePort, transportProtocol, serviceName string
	var speedLimit uint64 = 0

	if nodeInfoResponse.RawServerString == "" {
		return nil, fmt.Errorf("no server info in response")
	}
	if result := firstPortRe.FindStringSubmatch(nodeInfoResponse.RawServerString); len(result) > 1 {
		outsidePort = result[1]
	}
	if result := secondPortRe.FindStringSubmatch(nodeInfoResponse.RawServerString); len(result) > 1 {
		insidePort = result[1]
	}
	if result := hostRe.FindStringSubmatch(nodeInfoResponse.RawServerString); len(result) > 1 {
		host = result[1]
	}

	if insidePort != "" {
		p = insidePort
	} else {
		p = outsidePort
	}

	parsedPort, err := strconv.ParseInt(p, 10, 32)
	if err != nil {
		return nil, err
	}
	port := uint32(parsedPort)

	serverConf := strings.Split(nodeInfoResponse.RawServerString, ";")
	extraServerConf := strings.Split(serverConf[1], "|")
	transportProtocol = "tcp"
	serviceName = ""
	for _, item := range extraServerConf {
		conf := strings.Split(item, "=")
		key := conf[0]
		if key == "" {
			continue
		}
		value := conf[1]
		switch key {
		case "grpc":
			transportProtocol = "grpc"
		case "servicename":
			serviceName = value
		}
	}

	if c.SpeedLimit > 0 {
		speedLimit = uint64((c.SpeedLimit * 1000000) / 8)
	} else {
		speedLimit = uint64((nodeInfoResponse.SpeedLimit * 1000000) / 8)
	}
	// 转换为控制器统一使用的节点模型。
	nodeInfo := &api.NodeInfo{
		NodeType:          c.NodeType,
		NodeID:            c.NodeID,
		Port:              port,
		SpeedLimit:        speedLimit,
		TransportProtocol: transportProtocol,
		EnableTLS:         true,
		Host:              host,
		ServiceName:       serviceName,
	}

	return nodeInfo, nil
}

// ParseUserListResponse 转换用户列表，并计算本节点还能接受的设备数。
func (c *APIClient) ParseUserListResponse(userInfoResponse *[]UserResponse) (*[]api.UserInfo, error) {
	c.access.Lock()
	// 本轮计算结束后清空上次在线上报快照。
	defer func() {
		c.LastReportOnline = make(map[int]int)
		c.access.Unlock()
	}()

	var deviceLimit, localDeviceLimit = 0, 0
	var speedLimit uint64 = 0
	var userList []api.UserInfo
	for _, user := range *userInfoResponse {
		if c.DeviceLimit > 0 {
			deviceLimit = c.DeviceLimit
		} else {
			deviceLimit = user.DeviceLimit
		}

		// 面板统计包含所有后端；扣除其它后端占用后再决定是否加载该用户。
		if deviceLimit > 0 && user.AliveIP > 0 {
			lastOnline := 0
			if v, ok := c.LastReportOnline[user.ID]; ok {
				lastOnline = v
			}
			// 仍有设备名额时，只允许本节点使用剩余名额。
			if localDeviceLimit = deviceLimit - user.AliveIP + lastOnline; localDeviceLimit > 0 {
				deviceLimit = localDeviceLimit
				// 无剩余名额但本节点已有在线连接时，暂时保留这些连接。
			} else if lastOnline > 0 {
				deviceLimit = lastOnline
				// 没有名额且本节点也没有在线连接，不加载该用户。
			} else {
				continue
			}
		}

		if c.SpeedLimit > 0 {
			speedLimit = uint64((c.SpeedLimit * 1000000) / 8)
		} else {
			speedLimit = uint64((user.SpeedLimit * 1000000) / 8)
		}
		userList = append(userList, api.UserInfo{
			UID:         user.ID,
			UUID:        user.UUID,
			Passwd:      user.Passwd,
			SpeedLimit:  speedLimit,
			DeviceLimit: deviceLimit,
			Port:        user.Port,
			Method:      user.Method,
		})
	}

	return &userList, nil
}

// ParseSSPanelNodeInfo 解析 SSPanel 2021.11 及以上版本的 custom_config。
func (c *APIClient) ParseSSPanelNodeInfo(nodeInfoResponse *NodeInfoResponse) (*api.NodeInfo, error) {
	var (
		speedLimit             uint64 = 0
		enableTLS, enableVless bool
		alterID                uint16 = 0
		transportProtocol      string
	)

	// 新格式必须包含 custom_config。
	if len(nodeInfoResponse.CustomConfig) == 0 {
		return nil, errors.New("custom_config is empty, disable custom config")
	}

	nodeConfig := new(CustomConfig)
	err := json.Unmarshal(nodeInfoResponse.CustomConfig, nodeConfig)
	if err != nil {
		return nil, fmt.Errorf("custom_config format error: %v", err)
	}

	if c.SpeedLimit > 0 {
		speedLimit = uint64((c.SpeedLimit * 1000000) / 8)
	} else {
		speedLimit = uint64((nodeInfoResponse.SpeedLimit * 1000000) / 8)
	}

	parsedPort, err := strconv.ParseInt(nodeConfig.OffsetPortNode, 10, 32)
	if err != nil {
		return nil, err
	}

	port := uint32(parsedPort)

	switch c.NodeType {
	case "Shadowsocks":
		transportProtocol = "tcp"
	case "V2ray":
		transportProtocol = nodeConfig.Network

		tlsType := nodeConfig.Security
		if tlsType == "tls" || tlsType == "xtls" {
			enableTLS = true
		}

		if nodeConfig.EnableVless == "1" {
			enableVless = true
		}
	case "Trojan":
		enableTLS = true
		transportProtocol = "tcp"

		// Trojan 默认 TCP，面板明确下发时使用指定传输协议。
		if nodeConfig.Network != "" {
			transportProtocol = nodeConfig.Network
		}
	}

	// 转换面板下发的 REALITY 配置。
	realityConfig := new(api.REALITYConfig)
	if nodeConfig.RealityOpts != nil {
		r := nodeConfig.RealityOpts
		realityConfig = &api.REALITYConfig{
			Dest:             r.Dest,
			ProxyProtocolVer: r.ProxyProtocolVer,
			ServerNames:      r.ServerNames,
			PrivateKey:       r.PrivateKey,
			MinClientVer:     r.MinClientVer,
			MaxClientVer:     r.MaxClientVer,
			MaxTimeDiff:      r.MaxTimeDiff,
			ShortIds:         r.ShortIds,
		}
	}

	// 转换为控制器统一使用的节点模型。
	nodeInfo := &api.NodeInfo{
		NodeType:          c.NodeType,
		NodeID:            c.NodeID,
		Port:              port,
		SpeedLimit:        speedLimit,
		AlterID:           alterID,
		TransportProtocol: transportProtocol,
		Host:              nodeConfig.Host,
		Path:              nodeConfig.Path,
		EnableTLS:         enableTLS,
		EnableVless:       enableVless,
		VlessFlow:         nodeConfig.Flow,
		CypherMethod:      nodeConfig.Method,
		ServiceName:       nodeConfig.Servicename,
		Header:            nodeConfig.Header,
		EnableREALITY:     nodeConfig.EnableREALITY,
		REALITYConfig:     realityConfig,
	}

	return nodeInfo, nil
}

// compareVersion 比较点分数字版本：大于返回 1，小于返回 -1，相等返回 0。
func compareVersion(version1, version2 string) int {
	n, m := len(version1), len(version2)
	i, j := 0, 0
	for i < n || j < m {
		x := 0
		for ; i < n && version1[i] != '.'; i++ {
			x = x*10 + int(version1[i]-'0')
		}
		i++ // 跳过点号
		y := 0
		for ; j < m && version2[j] != '.'; j++ {
			y = y*10 + int(version2[j]-'0')
		}
		j++ // 跳过点号
		if x > y {
			return 1
		}
		if x < y {
			return -1
		}
	}
	return 0
}
