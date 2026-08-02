package model

import (
	"database/sql/driver"
	"fmt"
	"time"
)

type DateTime time.Time

func (t DateTime) MarshalJSON() ([]byte, error) {
	s := time.Time(t).Format("2006-01-02 15:04:05")
	return []byte(fmt.Sprintf(`"%s"`, s)), nil
}

func (t *DateTime) UnmarshalJSON(data []byte) error {
	parsed, err := time.ParseInLocation(`"2006-01-02 15:04:05"`, string(data), time.Local)
	if err != nil {
		return err
	}
	*t = DateTime(parsed)
	return nil
}

func (DateTime) GormDataType() string { return "datetime" }

func (t DateTime) Value() (driver.Value, error) {
	return time.Time(t), nil
}

func (t *DateTime) Scan(v interface{}) error {
	switch val := v.(type) {
	case time.Time:
		*t = DateTime(val)
		return nil
	}
	return fmt.Errorf("cannot scan %T into DateTime", v)
}
