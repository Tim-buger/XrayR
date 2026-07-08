// Package mydispatcher 定制 Dispatcher：实现限速与在线设备统计
package mydispatcher

import "github.com/xtls/xray-core/features/routing"

//go:generate go run github.com/xtls/xray-core/common/errors/errorgen

// Type returns the routing dispatcher feature token.
func Type() interface{} {
	return routing.DispatcherType()
}
