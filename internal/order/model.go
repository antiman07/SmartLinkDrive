package order

import "time"

// Status 订单状态枚举（持久化为字符串）。
type Status string

const (
	StatusCreated   Status = "created"   // 已创建，待调度
	StatusAssigned  Status = "assigned"  // 已分配车辆/司机，待接单
	StatusAccepted  Status = "accepted"  // 司机已接单，待出发/服务中
	StatusInService Status = "serving"   // 服务中（行程进行中）
	StatusCompleted Status = "completed" // 已完成
	StatusCanceled  Status = "canceled"  // 已取消（乘客/司机/系统）
)

// Order 订单 GORM 模型（最小可用版本）。
// 后续可按需要补充字段（价格明细、轨迹 ID 等）。
type Order struct {
	ID string `gorm:"primaryKey;size:36"`

	// 业务关联
	UserID    string `gorm:"index;size:36;not null"`          // 下单用户
	VehicleID string `gorm:"index;size:36"`                   // 关联车辆
	DriverID  string `gorm:"index;size:36"`                   // 司机/运营人员
	Status    Status `gorm:"type:varchar(16);index;not null"` // 当前状态
	BizTag    string `gorm:"size:32"`                         // 可选：业务标签，如 city_id、业务线等
	Channel   string `gorm:"size:32"`                         // 下单渠道（app、小程序、运营后台等）

	// 起终点信息（简化为字符串，后续可抽为结构）
	PickupAddress  string `gorm:"size:255"`
	DropoffAddress string `gorm:"size:255"`

	// 金额信息（单位：分）
	EstimatedPrice int64  `gorm:"not null;default:0"` // 预估费用
	FinalPrice     int64  `gorm:"not null;default:0"` // 实际费用（完成后写入）
	Currency       string `gorm:"size:8;not null;default:'CNY'"`

	// 时间信息
	CreatedAt   time.Time  `gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime"`
	AcceptedAt  *time.Time // 接单时间
	StartedAt   *time.Time // 行程开始时间
	CompletedAt *time.Time // 完成时间
	CanceledAt  *time.Time // 取消时间
}
