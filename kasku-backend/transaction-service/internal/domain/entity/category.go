package entity

import (
	"time"

	"github.com/google/uuid"
)

type CategoryType string

const (
	CategoryIncome  CategoryType = "INCOME"
	CategoryExpense CategoryType = "EXPENSE"
	CategoryBoth    CategoryType = "BOTH"
)

func (t CategoryType) IsValid() bool {
	switch t {
	case CategoryIncome, CategoryExpense, CategoryBoth:
		return true
	default:
		return false
	}
}

type Category struct {
	ID           uuid.UUID    `json:"id"`
	Name         string       `json:"name"`
	Icon         string       `json:"icon"`
	Color        string       `json:"color"`
	CategoryType CategoryType `json:"category_type"`
	IsDefault    bool         `json:"is_default"`
	IsDeleted    bool         `json:"is_deleted"`
	DeletedAt    *time.Time   `json:"deleted_at,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}
