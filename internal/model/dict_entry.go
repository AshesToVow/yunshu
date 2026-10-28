package model

import (
	"time"

	"gorm.io/gorm"
)

// DictEntry 数据字典条目。
// value 使用 MEDIUMTEXT 且不参与索引，避免 VARCHAR(4096) 无法容纳完整 kubeconfig（含证书 base64）；
// 唯一性由 (dict_type, value_hash) 与 (dict_type, label) 保证。
// ActiveKey 解决软删除与唯一索引的冲突：活跃行恒为 0，软删除后写入自身 id（非 0 互不相同）。
type DictEntry struct {
	ID        uint           `json:"id" gorm:"primaryKey;comment:主键ID"`
	DictType  string         `json:"dict_type" gorm:"size:64;not null;index:idx_dict_type_status,priority:1;uniqueIndex:idx_dict_type_value_uq,priority:1;uniqueIndex:idx_dict_type_label_uq,priority:1;comment:字典类型"`
	Label     string         `json:"label" gorm:"size:128;not null;uniqueIndex:idx_dict_type_label_uq,priority:2;comment:显示标签"`
	Value     string         `json:"value" gorm:"type:text;not null;comment:字典值（如完整 kubeconfig）"`
	ValueHash string         `json:"-" gorm:"size:64;not null;default:'';uniqueIndex:idx_dict_type_value_uq,priority:2;comment:字典值 SHA-256（唯一性校验）"`
	ActiveKey uint           `json:"-" gorm:"not null;default:0;uniqueIndex:idx_dict_type_value_uq,priority:3;uniqueIndex:idx_dict_type_label_uq,priority:3;comment:活跃唯一键：0=活跃，软删除后置为 id"`
	Sort      int            `json:"sort" gorm:"not null;default:0;comment:排序"`
	Status    int            `json:"status" gorm:"not null;default:1;index:idx_dict_type_status,priority:2;comment:状态 1启用 0停用"`
	Remark    string         `json:"remark" gorm:"size:512;comment:备注"`
	CreatedAt time.Time      `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"comment:更新时间"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;index:idx_dict_type_status,priority:3;comment:删除时间"`
}

// TableName 指定 GORM 表名。
func (DictEntry) TableName() string {
	return "dict_entries"
}
