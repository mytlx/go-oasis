package util

import "time"

// MillisToTime 将毫秒级 Unix 时间戳 (int64) 转换为 time.Time
func MillisToTime(ms int64) time.Time {
	if ms <= 0 {
		return time.Time{} // 返回零值时间，防止负数或零导致 time.Unix 异常
	}
	// 1. ms / 1000 得到秒数
	// 2. (ms % 1000) * 1000000 得到剩余的毫秒数对应的纳秒数
	return time.Unix(ms/1000, (ms%1000)*1000000)
}
