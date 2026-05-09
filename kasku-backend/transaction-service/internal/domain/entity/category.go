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

type Category struct {
	ID           uuid.UUID
	Name         string
	Icon         string
	Color        string
	CategoryType CategoryType
	IsDefault    bool
	IsDeleted    bool
	DeletedAt    *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
