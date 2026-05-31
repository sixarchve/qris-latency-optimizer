package usecase

import (
	"encoding/json"
	"errors"
	"qris-latency-optimizer/domain/entity"
	"qris-latency-optimizer/domain/repository"
	"qris-latency-optimizer/internal/qris"
	"qris-latency-optimizer/repository/rabbitmq"
	"qris-latency-optimizer/repository/redis"
	"time"

	"github.com/google/uuid"
)

type TransactionUsecase interface {
	ScanQR(req entity.ScanQRRequest) (*entity.TransactionResponse, error)
	ConfirmPaymentAsync(transactionIDStr string) error
	ConfirmPaymentSync(transactionIDStr string) (*entity.TransactionResponse, error)
	GetTransactionStatus(transactionIDStr string) (*entity.TransactionResponse, error)
}

type transactionUsecase struct {
	txRepo       repository.TransactionRepository
	merchantRepo repository.MerchantRepository
}

func NewTransactionUsecase(txRepo repository.TransactionRepository, merchantRepo repository.MerchantRepository) TransactionUsecase {
	return &transactionUsecase{txRepo: txRepo, merchantRepo: merchantRepo}
}

func (u *transactionUsecase) ScanQR(req entity.ScanQRRequest) (*entity.TransactionResponse, error) {
	var merchant *entity.Merchant

	merchantUUID, err := uuid.Parse(req.MerchantID)
	if err == nil {
		merchant, err = u.merchantRepo.FindByID(merchantUUID)
		if err != nil {
			return nil, errors.New("merchant not found")
		}
	} else {
		// cache lookup
		if cached, ok := redis.GetMerchant(req.MerchantID); ok {
			merchant = cached
		} else {
			merchant, err = u.merchantRepo.FindByQRID(req.MerchantID)
			if err != nil {
				return nil, errors.New("merchant not found")
			}
			redis.CacheMerchant(*merchant)
		}
	}

	qrMerchantID, qrAmount, err := qris.ParsePayload(req.QRPayload)
	if err != nil {
		return nil, err
	}
	if qrMerchantID != merchant.QRID {
		return nil, errors.New("qr payload merchant does not match merchant id")
	}
	if float64(qrAmount) != req.Amount {
		return nil, errors.New("qr payload amount does not match amount")
	}

	tx := entity.Transaction{
		ID:         uuid.New(),
		MerchantID: merchant.ID,
		Amount:     req.Amount,
		Status:     "PENDING",
		CreatedAt:  time.Now(),
	}

	if err := u.txRepo.Create(&tx); err != nil {
		return nil, errors.New("failed to create transaction")
	}

	redis.CacheTransaction(tx)

	return &entity.TransactionResponse{
		TransactionID: tx.ID.String(),
		MerchantID:    merchant.ID.String(),
		Amount:        tx.Amount,
		Status:        tx.Status,
		CreatedAt:     tx.CreatedAt,
		CachedFrom:    false,
	}, nil
}

func (u *transactionUsecase) ConfirmPaymentAsync(transactionIDStr string) error {
	if _, err := uuid.Parse(transactionIDStr); err != nil {
		return errors.New("invalid transaction id")
	}

	event := map[string]string{
		"transaction_id": transactionIDStr,
	}
	eventJSON, _ := json.Marshal(event)

	err := rabbitmq.PublishMessage(string(eventJSON))
	if err != nil {
		return errors.New("failed to queue transaction: " + err.Error())
	}
	return nil
}

func (u *transactionUsecase) ConfirmPaymentSync(transactionIDStr string) (*entity.TransactionResponse, error) {
	if _, err := uuid.Parse(transactionIDStr); err != nil {
		return nil, errors.New("invalid transaction id")
	}

	rows, err := u.txRepo.UpdateStatus(transactionIDStr, "SUCCESS")
	if err != nil {
		return nil, errors.New("failed to confirm payment")
	}
	if rows == 0 {
		return nil, errors.New("transaction not found")
	}

	redis.DeleteTransaction(transactionIDStr)

	tx, err := u.txRepo.FindByID(transactionIDStr)
	if err != nil {
		return nil, errors.New("transaction not found")
	}

	go func() {
		_ = rabbitmq.PublishNotification(
			tx.ID.String(),
			tx.MerchantID.String(),
			tx.Merchant.MerchantName,
			tx.Amount,
		)
	}()

	return &entity.TransactionResponse{
		TransactionID: tx.ID.String(),
		MerchantID:    tx.MerchantID.String(),
		Amount:        tx.Amount,
		Status:        tx.Status,
		CreatedAt:     tx.CreatedAt,
	}, nil
}

func (u *transactionUsecase) GetTransactionStatus(transactionIDStr string) (*entity.TransactionResponse, error) {
	if _, err := uuid.Parse(transactionIDStr); err != nil {
		return nil, errors.New("invalid transaction id")
	}

	// Cache check
	if tx, ok := redis.GetTransaction(transactionIDStr); ok {
		return &entity.TransactionResponse{
			TransactionID: tx.ID.String(),
			MerchantID:    tx.MerchantID.String(),
			Amount:        tx.Amount,
			Status:        tx.Status,
			CreatedAt:     tx.CreatedAt,
			CachedFrom:    true,
		}, nil
	}

	tx, err := u.txRepo.FindByID(transactionIDStr)
	if err != nil {
		return nil, errors.New("transaction not found")
	}

	redis.CacheTransaction(*tx)

	return &entity.TransactionResponse{
		TransactionID: tx.ID.String(),
		MerchantID:    tx.MerchantID.String(),
		Amount:        tx.Amount,
		Status:        tx.Status,
		CreatedAt:     tx.CreatedAt,
		CachedFrom:    false,
	}, nil
}
