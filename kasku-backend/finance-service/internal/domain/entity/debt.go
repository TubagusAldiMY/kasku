package entity

import (
	"time"

	"github.com/google/uuid"
)

type DebtDirection string

const (
	DebtDirectionReceivable DebtDirection = "RECEIVABLE" // orang berhutang ke saya (piutang)
	DebtDirectionPayable    DebtDirection = "PAYABLE"    // saya berhutang ke orang
)

type DebtStatus string

const (
	DebtStatusActive  DebtStatus = "ACTIVE"
	DebtStatusSettled DebtStatus = "SETTLED"
)

type Debt struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	Direction       DebtDirection
	PersonName      string
	TotalAmount     int64 // nominal awal dalam IDR
	RemainingAmount int64 // sisa yang belum dilunasi
	DueDate         *time.Time
	Notes           string
	Status          DebtStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type DebtPayment struct {
	ID          uuid.UUID
	DebtID      uuid.UUID
	Amount      int64
	PaymentDate time.Time
	Notes       string
	CreatedAt   time.Time
}
