package order

import (
	"fmt"
	"time"
)

// AllowTransition 定义订单状态机的允许流转关系。
// 目前采用“有向图”方式进行配置，后续可根据需要抽到配置中心。
var AllowTransition = map[Status][]Status{
	StatusCreated:   {StatusAssigned, StatusCanceled},
	StatusAssigned:  {StatusAccepted, StatusCanceled},
	StatusAccepted:  {StatusInService, StatusCanceled},
	StatusInService: {StatusCompleted, StatusCanceled},
	// 终态：不允许从 completed / canceled 再流转
	StatusCompleted: {},
	StatusCanceled:  {},
}

// CanTransition 判断 from -> to 是否是一个允许的状态流转。
func CanTransition(from, to Status) bool {
	if from == to {
		return true
	}
	allowed, ok := AllowTransition[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// ApplyTransition 对订单应用状态变更，并维护关键时间字段。
// 仅在 CanTransition 返回 true 时调用。
func ApplyTransition(o *Order, to Status, now time.Time) error {
	if o == nil {
		return fmt.Errorf("order is nil")
	}
	from := o.Status
	if !CanTransition(from, to) {
		return fmt.Errorf("invalid order status transition: %s -> %s", from, to)
	}

	o.Status = to

	switch to {
	case StatusAccepted:
		if o.AcceptedAt == nil {
			t := now
			o.AcceptedAt = &t
		}
	case StatusInService:
		if o.StartedAt == nil {
			t := now
			o.StartedAt = &t
		}
	case StatusCompleted:
		if o.CompletedAt == nil {
			t := now
			o.CompletedAt = &t
		}
	case StatusCanceled:
		if o.CanceledAt == nil {
			t := now
			o.CanceledAt = &t
		}
	}
	return nil
}
