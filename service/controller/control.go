package controller

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/session"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/features/inbound"
	"github.com/xtls/xray-core/features/outbound"
	"github.com/xtls/xray-core/features/policy"
	"github.com/xtls/xray-core/features/stats"
	"github.com/xtls/xray-core/proxy"
	"github.com/xtls/xray-core/transport"

	"github.com/XrayR-project/XrayR/api"
	"github.com/XrayR-project/XrayR/common/limiter"
)

func (c *Controller) removeInbound(tag string) error {
	// 按 tag 删除入站
	err := c.ibm.RemoveHandler(context.Background(), tag)
	return err
}

// statsOutboundWrapper wraps outbound.Handler to ensure user downlink traffic is counted.
type statsOutboundWrapper struct {
	outbound.Handler
	pm      policy.Manager
	sm      stats.Manager
	limiter *limiter.Limiter
}

func WrapOutboundHandler(handler outbound.Handler, pm policy.Manager, sm stats.Manager, limiter *limiter.Limiter) outbound.Handler {
	if _, ok := handler.(*statsOutboundWrapper); ok {
		return handler
	}
	return &statsOutboundWrapper{Handler: handler, pm: pm, sm: sm, limiter: limiter}
}

func (w *statsOutboundWrapper) Dispatch(ctx context.Context, link *transport.Link) {
	// 禁用内核 splice，以防止 Vision/REALITY 绕过用户态统计路径。
	if sess := session.InboundFromContext(ctx); sess != nil {
		sess.CanSpliceCopy = 3
		if w.limiter != nil && sess.User != nil && sess.User.Email != "" {
			ip := sess.Source.Address.IP().String()
			if err := w.limiter.RecordOnlineIP(sess.Tag, sess.User.Email, ip); err != nil {
				log.Printf("Record online IP failed: tag=%s user=%s ip=%s err=%v", sess.Tag, sess.User.Email, ip, err)
			}
		}
	}
	w.Handler.Dispatch(ctx, link)
}

func (c *Controller) removeOutbound(tag string) error {
	// 按 tag 删除出站
	err := c.obm.RemoveHandler(context.Background(), tag)
	return err
}

func (c *Controller) addInbound(config *core.InboundHandlerConfig) error {
	// 将配置转为入站 handler 并注册到 xray-core
	rawHandler, err := core.CreateObject(c.server, config)
	if err != nil {
		return err
	}
	handler, ok := rawHandler.(inbound.Handler)
	if !ok {
		return fmt.Errorf("not an InboundHandler: %s", err)
	}
	if err := c.ibm.AddHandler(context.Background(), handler); err != nil {
		return err
	}
	return nil
}

func (c *Controller) addOutbound(config *core.OutboundHandlerConfig) error {
	// 将配置转为出站 handler 并注册到 xray-core
	rawHandler, err := core.CreateObject(c.server, config)
	if err != nil {
		return err
	}
	handler, ok := rawHandler.(outbound.Handler)
	if !ok {
		return fmt.Errorf("not an InboundHandler: %s", err)
	}
	// Wrap outbound handler to ensure downlink stats are always counted (e.g., REALITY/VLESS cases)
	handler = WrapOutboundHandler(handler, c.pm, c.stm, c.dispatcher.Limiter)
	if err := c.obm.AddHandler(context.Background(), handler); err != nil {
		return err
	}
	return nil
}

func (c *Controller) addUsers(users []*protocol.User, tag string) error {
	// 向指定入站添加用户
	handler, err := c.ibm.GetHandler(context.Background(), tag)
	if err != nil {
		return fmt.Errorf("no such inbound tag: %s", err)
	}
	inboundInstance, ok := handler.(proxy.GetInbound)
	if !ok {
		return fmt.Errorf("handler %s has not implemented proxy.GetInbound", tag)
	}

	userManager, ok := inboundInstance.GetInbound().(proxy.UserManager)
	if !ok {
		return fmt.Errorf("handler %s has not implemented proxy.UserManager", tag)
	}
	for _, item := range users {
		mUser, err := item.ToMemoryUser()
		if err != nil {
			return err
		}
		err = userManager.AddUser(context.Background(), mUser)
		if err != nil {
			return err
		}
		// Pre-register per-user traffic counters so core can increment them (downlink/uplink)
		uName := "user>>>" + mUser.Email + ">>>traffic>>>uplink"
		dName := "user>>>" + mUser.Email + ">>>traffic>>>downlink"
		if _, _ = stats.GetOrRegisterCounter(c.stm, uName); true {
			// no-op
		}
		if _, _ = stats.GetOrRegisterCounter(c.stm, dName); true {
			// no-op
		}
	}
	return nil
}

func (c *Controller) removeUsers(users []string, tag string) error {
	// 从指定入站移除用户
	handler, err := c.ibm.GetHandler(context.Background(), tag)
	if err != nil {
		return fmt.Errorf("no such inbound tag: %s", err)
	}
	inboundInstance, ok := handler.(proxy.GetInbound)
	if !ok {
		return fmt.Errorf("handler %s is not implement proxy.GetInbound", tag)
	}

	userManager, ok := inboundInstance.GetInbound().(proxy.UserManager)
	if !ok {
		return fmt.Errorf("handler %s is not implement proxy.UserManager", err)
	}
	for _, email := range users {
		err = userManager.RemoveUser(context.Background(), email)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) getTraffic(email string) (up int64, down int64, upCounter stats.Counter, downCounter stats.Counter) {
	// 读取单个用户的上下行统计
	upName := "user>>>" + email + ">>>traffic>>>uplink"
	downName := "user>>>" + email + ">>>traffic>>>downlink"
	upCounter = c.stm.GetCounter(upName)
	downCounter = c.stm.GetCounter(downName)
	if upCounter != nil && upCounter.Value() != 0 {
		up = upCounter.Value()
	} else {
		upCounter = nil
	}
	if downCounter != nil && downCounter.Value() != 0 {
		down = downCounter.Value()
	} else {
		downCounter = nil
	}
	return up, down, upCounter, downCounter
}

type trafficCounterSample struct {
	counter stats.Counter
	value   int64
}

func (c *Controller) resetTraffic(upCounterList *[]trafficCounterSample, downCounterList *[]trafficCounterSample) {
	// 只扣减本周期已上报的流量，避免把上报请求期间新增的流量清零。
	for _, upCounter := range *upCounterList {
		if upCounter.counter.Add(-upCounter.value) < 0 {
			upCounter.counter.Set(0)
		}
	}
	for _, downCounter := range *downCounterList {
		if downCounter.counter.Add(-downCounter.value) < 0 {
			downCounter.counter.Set(0)
		}
	}
}

func (c *Controller) AddInboundLimiter(tag string, nodeSpeedLimit uint64, userList *[]api.UserInfo, globalDeviceLimitConfig *limiter.GlobalDeviceLimitConfig) error {
	// 初始化限速器（含设备数限制）
	err := c.dispatcher.Limiter.AddInboundLimiter(tag, nodeSpeedLimit, userList, globalDeviceLimitConfig)
	return err
}

func (c *Controller) UpdateInboundLimiter(tag string, updatedUserList *[]api.UserInfo) error {
	// 更新限速器内的用户信息
	err := c.dispatcher.Limiter.UpdateInboundLimiter(tag, updatedUserList)
	return err
}

func (c *Controller) DeleteInboundLimiter(tag string) error {
	// 删除指定入站的限速器
	err := c.dispatcher.Limiter.DeleteInboundLimiter(tag)
	return err
}

func (c *Controller) GetOnlineDevice(tag string) (*[]api.OnlineUser, error) {
	// 获取在线设备列表
	return c.dispatcher.Limiter.GetOnlineDevice(tag)
}

func (c *Controller) UpdateRule(tag string, newRuleList []api.DetectRule) error {
	// 更新审计规则
	err := c.dispatcher.RuleManager.UpdateRule(tag, newRuleList)
	return err
}

func (c *Controller) GetDetectResult(tag string) (*[]api.DetectResult, error) {
	// 获取命中记录
	return c.dispatcher.RuleManager.GetDetectResult(tag)
}
