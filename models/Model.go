package models

import (
	"database/sql/driver"

	"time"

	"github.com/goccy/go-json"

	"backnet/components"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

type IntModel struct {
	Val   int64
	Valid bool
}
type FloatModel struct {
	Val   float64
	Valid bool
}
type BoolModel struct {
	Val   bool
	Valid bool
}
type ByteModel struct {
	Val   []byte
	Valid bool
}

type StringModel struct {
	Val   string
	Valid bool
}
type TimeDeleteModel struct {
	Val   time.Time
	Valid bool
}
type TimeModel struct {
	Val   time.Time
	Valid bool
}

func (n *IntModel) Scan(Value any) error {
	if Value == nil {
		n.Val, n.Valid = 0, false
		return nil
	}
	n.Valid = true
	return components.СonvertAssign(&n.Val, Value)
}

func (n IntModel) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Val, nil
}

func (n *IntModel) Set(Value int64) error {
	return n.Scan(Value)
}

func (n IntModel) Get() int64 {
	if !n.Valid {
		return 0
	}
	return n.Val
}

func (n IntModel) String() string {
	s := ""

	if n.Valid {
		components.СonvertAssign(&s, n.Val)
	}

	return s
}

func (n IntModel) MarshalJSON() ([]byte, error) {
	if n.Valid {
		return json.Marshal(n.Val)
	}
	return json.Marshal(nil)
}

func (n *IntModel) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		n.Valid = false
		return nil
	}
	err := json.Unmarshal(b, &n.Val)
	if err == nil {
		n.Valid = true
	}
	return err
}

func (n *FloatModel) Scan(Value any) error {
	if Value == nil {
		n.Val, n.Valid = 0, false
		return nil
	}
	n.Valid = true
	return components.СonvertAssign(&n.Val, Value)
}

func (n FloatModel) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Val, nil
}

func (n *FloatModel) Set(Value float64) error {
	return n.Scan(Value)
}

func (n FloatModel) Get() float64 {
	if !n.Valid {
		return 0
	}
	return n.Val
}

func (n FloatModel) String() string {
	s := ""

	if n.Valid {
		components.СonvertAssign(&s, n.Val)
	}

	return s
}

func (n FloatModel) MarshalJSON() ([]byte, error) {
	if n.Valid {
		return json.Marshal(n.Val)
	}
	return json.Marshal(nil)
}

func (n *FloatModel) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		n.Valid = false
		return nil
	}
	err := json.Unmarshal(b, &n.Val)
	if err == nil {
		n.Valid = true
	}
	return err
}

func (n *BoolModel) Scan(Value any) error {
	if Value == nil {
		n.Val, n.Valid = false, false
		return nil
	}
	n.Valid = true
	return components.СonvertAssign(&n.Val, Value)
}

func (n BoolModel) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Val, nil
}

func (n *BoolModel) Set(Value bool) error {
	return n.Scan(Value)
}

func (n BoolModel) Get() bool {
	if !n.Valid {
		return false
	}
	return n.Val
}

func (n BoolModel) String() string {
	s := ""

	if n.Valid {
		components.СonvertAssign(&s, n.Val)
	}

	return s
}

func (n BoolModel) MarshalJSON() ([]byte, error) {
	if n.Valid {
		return json.Marshal(n.Val)
	}
	return json.Marshal(nil)
}

func (n *BoolModel) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		n.Valid = false
		return nil
	}
	err := json.Unmarshal(b, &n.Val)
	if err == nil {
		n.Valid = true
	}
	return err
}

func (n *ByteModel) Scan(Value any) error {
	if Value == nil {
		n.Val, n.Valid = []byte{}, false
		return nil
	}
	n.Valid = true
	return components.СonvertAssign(&n.Val, Value)
}

func (n ByteModel) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Val, nil
}

func (n *ByteModel) Set(Value []byte) error {
	return n.Scan(Value)
}

func (n ByteModel) Get() []byte {
	if !n.Valid {
		return []byte{}
	}
	return n.Val
}

func (n ByteModel) String() string {
	s := ""

	if n.Valid {
		components.СonvertAssign(&s, n.Val)
	}

	return s
}

func (n ByteModel) MarshalJSON() ([]byte, error) {
	if n.Valid {
		return json.Marshal(n.Val)
	}
	return json.Marshal(nil)
}

func (n *ByteModel) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		n.Valid = false
		return nil
	}
	err := json.Unmarshal(b, &n.Val)
	if err == nil {
		n.Valid = true
	}
	return err
}

func (n *StringModel) Scan(Value any) error {
	if Value == nil {
		n.Val, n.Valid = "", false
		return nil
	}
	n.Valid = true
	return components.СonvertAssign(&n.Val, Value)
}

func (n StringModel) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Val, nil
}

func (n *StringModel) Set(Value string) error {
	return n.Scan(Value)
}

func (n StringModel) Get() string {
	if !n.Valid {
		return ""
	}
	return n.Val
}

func (n StringModel) String() string {
	if n.Valid {
		return n.Val
	}

	return ""
}

func (n StringModel) MarshalJSON() ([]byte, error) {
	if n.Valid {
		return json.Marshal(n.Val)
	}
	return json.Marshal(nil)
}

func (n *StringModel) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		n.Valid = false
		return nil
	}
	err := json.Unmarshal(b, &n.Val)
	if err == nil {
		n.Valid = true
	}
	return err
}

func (n *TimeModel) Scan(Value any) error {
	if Value == nil {
		n.Val, n.Valid = time.Time{}, false
		return nil
	}
	n.Valid = true
	return components.СonvertAssign(&n.Val, Value)
}

func (n TimeModel) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Val, nil
}

func (n *TimeModel) Set(Value time.Time) error {
	return n.Scan(Value)
}

func (n TimeModel) Get() time.Time {
	if !n.Valid {
		return time.Time{}
	}
	return n.Val
}

func (n TimeModel) String() string {
	s := ""

	if n.Valid {
		components.СonvertAssign(&s, n.Val)
	}

	return s
}

func (n TimeModel) MarshalJSON() ([]byte, error) {
	if n.Valid {
		return json.Marshal(n.Val)
	}
	return json.Marshal(nil)
}

func (n *TimeModel) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		n.Valid = false
		return nil
	}
	err := json.Unmarshal(b, &n.Val)
	if err == nil {
		n.Valid = true
	}
	return err
}

func (n *TimeDeleteModel) Scan(Value any) error {
	if Value == nil {
		n.Val, n.Valid = time.Time{}, false
		return nil
	}
	n.Valid = true
	return components.СonvertAssign(&n.Val, Value)
}

func (n TimeDeleteModel) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Val, nil
}

func (n TimeDeleteModel) MarshalJSON() ([]byte, error) {
	if n.Valid {
		return json.Marshal(n.Val)
	}
	return json.Marshal(nil)
}

func (n *TimeDeleteModel) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		n.Valid = false
		return nil
	}
	err := json.Unmarshal(b, &n.Val)
	if err == nil {
		n.Valid = true
	}
	return err
}

func (n *TimeDeleteModel) Set(Value time.Time) error {
	return n.Scan(Value)
}

func (n TimeDeleteModel) Get() time.Time {
	if !n.Valid {
		return time.Time{}
	}
	return n.Val
}

func (n TimeDeleteModel) String() string {
	s := ""

	if n.Valid {
		components.СonvertAssign(&s, n.Val)
	}

	return s
}

func (TimeDeleteModel) QueryClauses(f *schema.Field) []clause.Interface {
	return []clause.Interface{gorm.SoftDeleteQueryClause{Field: f}}
}

func (TimeDeleteModel) UpdateClauses(f *schema.Field) []clause.Interface {
	return []clause.Interface{gorm.SoftDeleteUpdateClause{Field: f}}
}

type Model struct {
	Id IntModel `gorm:"primarykey"`
}

func (n Model) Valid() bool {
	if n.Id.Valid {
		return true
	} else {
		return false
	}
}
