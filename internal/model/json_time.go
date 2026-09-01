package model

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// DateTime 自定义时间类型，用于统一 JSON 格式和数据库交互
// 这样写的好处就是 DateTime 和 time.Time 类型可以直接互转，类似 int32 和 int64 可以互转，但是 int32 和 string 就不能直接转
type DateTime time.Time

// MarshalJSON 触发时机：结构体转 JSON 时（如 c.JSON()、json.Marshal）
// 定义了 MarshalJSON，c.JSON() 执行时会自动判断有 MarshalJSON 会执行
// 定义能力，但是不关心谁调用
func (t DateTime) MarshalJSON() ([]byte, error) {
	s := time.Time(t).Format("2006-01-02 15:04:05")	// 把 DateTime 类型转成 time.Time，再调用 Format
	return []byte(fmt.Sprintf(`"%s"`, s)), nil		// []byte(字符串)：把字符串转换成对应的 UTF‑8 字节切片
}

// UnmarshalJSON 触发时机：JSON 解析到结构体时（如 c.ShouldBindJSON）
func (t *DateTime) UnmarshalJSON(data []byte) error {
	parsed, err := time.ParseInLocation(`"2006-01-02 15:04:05"`, string(data), time.Local)	// 解析时间 成 time.Time
	if err != nil {
		return err
	}
	*t = DateTime(parsed)		// 把 time.Time 类型转成 DateTime
	return nil
}

// GormDataType 触发时机：GORM 建表/迁移时，告诉数据库该字段用 datetime 类型
func (DateTime) GormDataType() string { return "datetime" }

// Value 触发时机：写入数据库时，GORM 调用此方法将 DateTime 转为数据库可接受的值
func (t DateTime) Value() (driver.Value, error) {
	return time.Time(t), nil		// 把 DateTime 类型转成 time.Time
}

// Scan 触发时机：从数据库读取时，GORM 调用此方法将数据库值转为 DateTime
func (t *DateTime) Scan(v interface{}) error {
	// 类型选择（type switch）
	// type switch 是 Go 中的语法结构，用于根据接口变量的具体类型执行不同的逻辑
	switch val := v.(type) {
	case time.Time:
		*t = DateTime(val)	// 把 time.Time 类型转成 DateTime
		return nil
	}
	return fmt.Errorf("cannot scan %T into DateTime", v)
}
