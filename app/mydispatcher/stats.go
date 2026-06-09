package mydispatcher

import (
	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/buf"
	"github.com/xtls/xray-core/features/stats"
)

type SizeStatWriter struct {
	// 对 Writer 包装以统计流量大小
	Counter stats.Counter
	Writer  buf.Writer
}

func (w *SizeStatWriter) WriteMultiBuffer(mb buf.MultiBuffer) error {
	// 累计流量并转发写入
	w.Counter.Add(int64(mb.Len()))
	return w.Writer.WriteMultiBuffer(mb)
}

func (w *SizeStatWriter) Close() error {
	// 关闭底层 writer
	return common.Close(w.Writer)
}

func (w *SizeStatWriter) Interrupt() {
	// 中断底层 writer
	common.Interrupt(w.Writer)
}
