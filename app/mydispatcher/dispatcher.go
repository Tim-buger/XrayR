// Package mydispatcher 定制 Dispatcher：实现限速与在线设备统计
package mydispatcher

//go:generate go run github.com/xtls/xray-core/common/errors/errorgen

// Type returns the feature type token for the custom dispatcher state.
func Type() interface{} {
	return (*DefaultDispatcher)(nil)
}
