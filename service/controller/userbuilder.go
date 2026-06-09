package controller

import (
	"fmt"
	"strings"

	"github.com/sagernet/sing-shadowsocks/shadowaead_2022"
	C "github.com/sagernet/sing/common"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/infra/conf"
	"github.com/xtls/xray-core/proxy/shadowsocks"
	"github.com/xtls/xray-core/proxy/shadowsocks_2022"
	"github.com/xtls/xray-core/proxy/trojan"
	"github.com/xtls/xray-core/proxy/vless"

	"github.com/XrayR-project/XrayR/api"
)

var AEADMethod = map[shadowsocks.CipherType]uint8{
	shadowsocks.CipherType_AES_128_GCM:        0,
	shadowsocks.CipherType_AES_256_GCM:        0,
	shadowsocks.CipherType_CHACHA20_POLY1305:  0,
	shadowsocks.CipherType_XCHACHA20_POLY1305: 0,
}

func (c *Controller) buildVmessUser(userInfo *[]api.UserInfo) (users []*protocol.User) {
	// 生成 vmess 用户列表
	users = make([]*protocol.User, len(*userInfo))
	for i, user := range *userInfo {
		vmessAccount := &conf.VMessAccount{
			ID:       user.UUID,
			Security: "auto",
		}
		users[i] = &protocol.User{
			Level:   0,
			Email:   c.buildUserTag(&user), // Email: InboundTag|email|uid
			Account: serial.ToTypedMessage(vmessAccount.Build()),
		}
	}
	return users
}

func (c *Controller) buildVlessUser(userInfo *[]api.UserInfo) (users []*protocol.User) {
	// 生成 vless 用户列表
	users = make([]*protocol.User, len(*userInfo))
	for i, user := range *userInfo {
		vlessAccount := &vless.Account{
			Id:   user.UUID,
			Flow: c.nodeInfo.VlessFlow,
		}
		users[i] = &protocol.User{
			Level:   0,
			Email:   c.buildUserTag(&user),
			Account: serial.ToTypedMessage(vlessAccount),
		}
	}
	return users
}

func (c *Controller) buildTrojanUser(userInfo *[]api.UserInfo) (users []*protocol.User) {
	// 生成 trojan 用户列表
	users = make([]*protocol.User, len(*userInfo))
	for i, user := range *userInfo {
		trojanAccount := &trojan.Account{
			Password: user.UUID,
		}
		users[i] = &protocol.User{
			Level:   0,
			Email:   c.buildUserTag(&user),
			Account: serial.ToTypedMessage(trojanAccount),
		}
	}
	return users
}

func (c *Controller) buildSSUser(userInfo *[]api.UserInfo, method string) (users []*protocol.User) {
	// 生成 shadowsocks 用户列表
	users = make([]*protocol.User, len(*userInfo))

	for i, user := range *userInfo {
		// Shadowsocks 2022 使用 base64 密钥，多用户账号不再单独指定加密方法。
		if C.Contains(shadowaead_2022.List, strings.ToLower(method)) {
			e := c.buildUserTag(&user)
			users[i] = &protocol.User{
				Level: 0,
				Email: e,
				Account: serial.ToTypedMessage(&shadowsocks_2022.Account{
					Key: user.Passwd,
				}),
			}
		} else {
			users[i] = &protocol.User{
				Level: 0,
				Email: c.buildUserTag(&user),
				Account: serial.ToTypedMessage(&shadowsocks.Account{
					Password:   user.Passwd,
					CipherType: cipherFromString(method),
				}),
			}
		}
	}
	return users
}

func (c *Controller) buildSSPluginUser(userInfo *[]api.UserInfo) (users []*protocol.User) {
	// 生成 shadowsocks 插件用户列表
	users = make([]*protocol.User, len(*userInfo))

	for i, user := range *userInfo {
		// Shadowsocks 2022 使用 base64 密钥，多用户账号不再单独指定加密方法。
		if C.Contains(shadowaead_2022.List, strings.ToLower(user.Method)) {
			e := c.buildUserTag(&user)
			users[i] = &protocol.User{
				Level: 0,
				Email: e,
				Account: serial.ToTypedMessage(&shadowsocks_2022.Account{
					Key: user.Passwd,
				}),
			}
		} else {
			// 只有受支持的 AEAD 加密方法才能创建传统 Shadowsocks 用户。
			cypherMethod := cipherFromString(user.Method)
			if _, ok := AEADMethod[cypherMethod]; ok {
				users[i] = &protocol.User{
					Level: 0,
					Email: c.buildUserTag(&user),
					Account: serial.ToTypedMessage(&shadowsocks.Account{
						Password:   user.Passwd,
						CipherType: cypherMethod,
					}),
				}
			}
		}
	}
	return users
}

func cipherFromString(c string) shadowsocks.CipherType {
	// cipher 字符串映射到枚举
	switch strings.ToLower(c) {
	case "aes-128-gcm", "aead_aes_128_gcm":
		return shadowsocks.CipherType_AES_128_GCM
	case "aes-256-gcm", "aead_aes_256_gcm":
		return shadowsocks.CipherType_AES_256_GCM
	case "chacha20-poly1305", "aead_chacha20_poly1305", "chacha20-ietf-poly1305":
		return shadowsocks.CipherType_CHACHA20_POLY1305
	case "none", "plain":
		return shadowsocks.CipherType_NONE
	default:
		return shadowsocks.CipherType_UNKNOWN
	}
}

func (c *Controller) buildUserTag(user *api.UserInfo) string {
	// 用户标识：InboundTag|email|uid
	return fmt.Sprintf("%s|%s|%d", c.Tag, user.Email, user.UID)
}
