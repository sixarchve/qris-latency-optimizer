package models

import (
	"time"

	"github.com/google/uuid"
)

type Transaction struct {
    ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
    MerchantID uuid.UUID `gorm:"column:merchant_id;type:uuid;index"`
    Amount     float64   `gorm:"type:decimal(15,2);not null"`
    Status     string    `gorm:"type:varchar(20);default:'PENDING'"`
    CreatedAt  time.Time `gorm:"autoCreateTime"`

    Merchant   Merchant  `gorm:"foreignKey:MerchantID"`
}