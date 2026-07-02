package rule

import (
	"fmt"

	"github.com/libtnb/validator"
	"gorm.io/gorm"
)

// Exists 验证一个值在某个表中的字段中存在，支持同时判断多个字段
// 用法：exists:表名称,字段名称,字段名称
// 例子：exists:users,phone,email
type Exists struct {
	db *gorm.DB
}

func NewExists(db *gorm.DB) *Exists {
	return &Exists{db: db}
}

func (r *Exists) Signature() string { return "exists" }

func (r *Exists) Message() string { return "{field} is not exists" }

func (r *Exists) Passes(f validator.Field) bool {
	rv := f.Val()
	if validator.IsEmptyValue(rv) {
		return true
	}
	args := f.Attrs()
	if len(args) < 2 {
		return false
	}

	val := rv.Interface()
	tableName := args[0]
	fieldNames := args[1:]

	query := r.db.Table(tableName).Where(fmt.Sprintf("%s = ?", fieldNames[0]), val)
	for _, fieldName := range fieldNames[1:] {
		query = query.Or(fmt.Sprintf("%s = ?", fieldName), val)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false
	}

	return count != 0
}
