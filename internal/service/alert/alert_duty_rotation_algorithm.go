package alert

import "time"

// AlertDutyRotationSlot 轮换中的单个班次片段（区间已裁剪到查询窗口）。
type AlertDutyRotationSlot struct {
	Index    int64
	StartsAt time.Time
	EndsAt   time.Time
	UserIDs  []uint
}

func floorDiv(a, b int64) int64 {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

func floorMod(a, b int64) int64 {
	m := a % b
	if m < 0 {
		m += b
	}
	return m
}

// ComputeRotationSlots 按「起算时间 + 固定周期 + 参与人顺序」计算 [from, to) 内的班次。
// 支持 from 早于起算时间（负数下标取模回落），返回区间已裁剪到 [from, to)。
func ComputeRotationSlots(anchor time.Time, period time.Duration, participants []uint, from, to time.Time) []AlertDutyRotationSlot {
	if period <= 0 || len(participants) == 0 || !to.After(from) {
		return nil
	}
	periodNanos := int64(period)
	n := int64(len(participants))
	index := floorDiv(int64(from.Sub(anchor)), periodNanos)
	slotStart := anchor.Add(time.Duration(index) * period)
	var out []AlertDutyRotationSlot
	for slotStart.Before(to) {
		slotEnd := slotStart.Add(period)
		if slotEnd.After(from) {
			start := slotStart
			if start.Before(from) {
				start = from
			}
			end := slotEnd
			if end.After(to) {
				end = to
			}
			out = append(out, AlertDutyRotationSlot{
				Index:    index,
				StartsAt: start,
				EndsAt:   end,
				UserIDs:  []uint{participants[floorMod(index, n)]},
			})
		}
		index++
		slotStart = slotEnd
	}
	return out
}

// CurrentRotationSlot 返回 at 时刻所处的班次（不做窗口裁剪）。
func CurrentRotationSlot(anchor time.Time, period time.Duration, participants []uint, at time.Time) (AlertDutyRotationSlot, bool) {
	if period <= 0 || len(participants) == 0 {
		return AlertDutyRotationSlot{}, false
	}
	index := floorDiv(int64(at.Sub(anchor)), int64(period))
	start := anchor.Add(time.Duration(index) * period)
	n := int64(len(participants))
	return AlertDutyRotationSlot{
		Index:    index,
		StartsAt: start,
		EndsAt:   start.Add(period),
		UserIDs:  []uint{participants[floorMod(index, n)]},
	}, true
}
