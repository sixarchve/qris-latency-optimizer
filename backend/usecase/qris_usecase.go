package usecase

import (
	"errors"
	"qris-latency-optimizer/domain/entity"
	"qris-latency-optimizer/domain/repository"
	"qris-latency-optimizer/internal/qris"
	"qris-latency-optimizer/repository/redis"
	"strconv"

	"github.com/google/uuid"
)

type QRISUsecase interface {
	GenerateQRIS(merchantIDStr string, amountStr string) (string, *entity.Merchant, int, error)
}

type qrisUsecase struct {
	repo repository.MerchantRepository
}

func NewQRISUsecase(repo repository.MerchantRepository) QRISUsecase {
	return &qrisUsecase{repo: repo}
}

func (u *qrisUsecase) GenerateQRIS(merchantIDStr string, amountStr string) (string, *entity.Merchant, int, error) {
	amount, err := strconv.Atoi(amountStr)
	if err != nil || amount <= 0 {
		return "", nil, 0, errors.New("invalid amount")
	}

	if merchantIDStr == "" {
		return "", nil, 0, errors.New("merchant_id is required")
	}

	merchantUUID, err := uuid.Parse(merchantIDStr)
	if err != nil {
		return "", nil, 0, errors.New("invalid merchant_id")
	}

	merchant, err := u.repo.FindByID(merchantUUID)
	if err != nil {
		return "", nil, 0, errors.New("merchant not found")
	}

	// Cache operations
	redis.CacheMerchant(*merchant)
	go redis.PrefetchRelatedMerchants(merchant.QRID)

	payload, err := qris.GeneratePayload(amount, merchant.MerchantName, merchant.QRID)
	if err != nil {
		return "", nil, 0, err
	}

	return payload, merchant, amount, nil
}
