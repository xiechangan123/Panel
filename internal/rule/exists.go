package rule

import (
	"github.com/leonelquinteros/gotext"
	"github.com/libtnb/validator"
	"gorm.io/gorm"
)

// Exists 验证一个值在某个表中的字段中存在，支持同时判断多个字段
// 用法：exists:表名称,字段名称,字段名称
// 例子：exists:users,phone,email
type Exists struct {
	db *gorm.DB
	t  *gotext.Locale
}

func NewExists(db *gorm.DB, t *gotext.Locale) *Exists {
	return &Exists{db: db, t: t}
}

func (r *Exists) Signature() string { return "exists" }

func (r *Exists) Message() string { return r.t.Get("{field} does not exist") }

func (r *Exists) Validate(f *validator.Field) (bool, error) {
	rv := f.Reflect()
	if validator.IsEmptyValue(rv) {
		return true, nil
	}
	args := f.Attrs()
	if len(args) < 2 {
		return false, nil
	}

	val := rv.Interface()
	tableName := args[0]
	fieldNames := args[1:]

	query := r.db.Table(tableName).Where(fieldNames[0]+" = ?", val)
	for _, fieldName := range fieldNames[1:] {
		query = query.Or(fieldName+" = ?", val)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}

	return count != 0, nil
}
