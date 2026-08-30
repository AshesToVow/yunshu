package model

import (
	"time"

	"gorm.io/gorm"
)

// InspectFinding 巡检风险台账：一次巡检产生的一条风险条目。
//
// 与 InspectRun 的区别：InspectRun 存「本期整体结论」，InspectFinding 存「逐条风险」，
// 是期次对比（新增/持续/已恢复）与整改闭环（责任人/期限）的数据基础。
// 明细样本仍不入库（留在报告文件），此表只存聚合后的风险条目，量级为每次巡检数十条。
type InspectFinding struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	ProjectID uint `json:"project_id" gorm:"not null;index:idx_inspect_finding_project_run,priority:1;comment:项目ID"`
	RunID     uint `json:"run_id" gorm:"not null;index:idx_inspect_finding_project_run,priority:2;comment:巡检执行ID"`

	// Fingerprint 跨期次识别同一条风险（type+name 归一化哈希），期次对比的连接键。
	Fingerprint string `json:"fingerprint" gorm:"size:64;not null;index:idx_inspect_finding_fp,priority:2;comment:风险指纹"`
	// Seq 本期台账内的展示序号，从 1 开始，用于报告中的「风险编号」。
	Seq int `json:"seq" gorm:"not null;default:0;comment:本期台账序号"`

	Type     string `json:"type" gorm:"size:200;not null;comment:巡检分类"`
	Name     string `json:"name" gorm:"size:200;not null;comment:检查项名称"`
	Severity string `json:"severity" gorm:"size:32;not null;index;comment:严重度 critical/warning"`
	Count    int    `json:"count" gorm:"not null;default:0;comment:受影响样本数"`

	// AffectedService 从样本标签推导的受影响业务/服务，把指标翻译成客户能读懂的对象。
	AffectedService string `json:"affected_service" gorm:"size:255;comment:受影响服务"`
	// Instances 受影响实例列表（逗号分隔，超出上限截断并以 … 结尾）。
	Instances string `json:"instances" gorm:"type:text;comment:受影响实例"`
	// Phenomenon 现象描述（当前值/阈值等客观事实）。
	Phenomenon string `json:"phenomenon" gorm:"type:text;comment:现象"`
	// Impact 业务影响说明。
	Impact string `json:"impact" gorm:"type:text;comment:业务影响"`
	// Suggestion 建议动作。
	Suggestion string `json:"suggestion" gorm:"type:text;comment:建议动作"`

	// State: new | persisting | recovered
	// new=本期新增，persisting=较上期持续存在，recovered=上期存在本期已恢复。
	State string `json:"state" gorm:"size:32;not null;default:new;index;comment:期次状态"`

	// Owner/DueDate 整改闭环字段。同一 Fingerprint 持续存在时从上一期沿用，
	// 避免每次巡检把人工填写的责任人冲掉。
	Owner   string     `json:"owner" gorm:"size:64;comment:责任人"`
	DueDate *time.Time `json:"due_date" gorm:"comment:整改期限"`

	// FirstSeenRunID/FirstSeenAt 首次发现的期次与时间，用于呈现「已持续 N 天」。
	FirstSeenRunID uint      `json:"first_seen_run_id" gorm:"not null;default:0;comment:首次发现的巡检ID"`
	FirstSeenAt    time.Time `json:"first_seen_at" gorm:"comment:首次发现时间"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (InspectFinding) TableName() string {
	return "inspect_findings"
}
