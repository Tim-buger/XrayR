// Package mydispatcher 定制 Dispatcher：实现限速与在线设备统计
package mydispatcher

//go:generate go run github.com/xtls/xray-core/common/errors/errorgen

// Type returns the feature type token for the custom dispatcher feature.
// It intentionally differs from routing.DispatcherType() to avoid replacing
// the core dispatcher and causing type assertion panics in xray-core.
func Type() interface{} {
	// Consumers should use server.GetFeature(mydispatcher.Type()) to access it.
	return (*DefaultDispatcher)(nil)
}
