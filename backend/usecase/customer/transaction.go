package customer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"qris-latency-optimizer/models"
	"qris-latency-optimizer/repository/database"
	"qris-latency-optimizer/repository/redis"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ScanQR - endpoint untuk scan QR dari customer
func ScanQR(c *gin.Context) {
	var req models.ScanQRRequest

	// Parse request body
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request: " + err.Error(),
		})
		return
	}

	// Generate transaction ID
	transactionID := uuid.New().String()
	cacheKey := fmt.Sprintf("transaction:%s", transactionID)

	// Buat transaction model
	transaction := models.Transaction{
		ID:         uuid.MustParse(transactionID),
		MerchantID: uuid.MustParse(req.MerchantID),
		Amount:     req.Amount,
		Status:     "PENDING",
		CreatedAt:  time.Now(),
	}

	// Simpan ke database
	if err := database.DB.Create(&transaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create transaction: " + err.Error(),
		})
		return
	}

	// Simpan ke Redis dengan TTL 10 menit
	transactionJSON, _ := json.Marshal(transaction)
	redis.Set(cacheKey, string(transactionJSON), 10*time.Minute)

	response := models.TransactionResponse{
		TransactionID: transactionID,
		MerchantID:    req.MerchantID,
		Amount:        req.Amount,
		Status:        "PENDING",
		CreatedAt:     transaction.CreatedAt,
		CachedFrom:    false,
	}

	c.JSON(http.StatusCreated, gin.H{
		"data":    response,
		"message": "transaction created successfully",
	})
}

// ConfirmPayment - endpoint untuk confirm pembayaran
func ConfirmPayment(c *gin.Context) {
	transactionID := c.Param("id")

	// Validasi UUID
	if _, err := uuid.Parse(transactionID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid transaction id",
		})
		return
	}

	cacheKey := fmt.Sprintf("transaction:%s", transactionID)

	// Update di database
	if err := database.DB.Model(&models.Transaction{}).
		Where("id = ?", transactionID).
		Update("status", "SUCCESS").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to confirm payment: " + err.Error(),
		})
		return
	}

	// Hapus dari cache (invalidate)
	redis.Delete(cacheKey)
	fmt.Println("✓ Cache invalidated after payment confirmation")

	// Ambil data transaksi yang sudah updated
	var transaction models.Transaction
	database.DB.First(&transaction, "id = ?", transactionID)

	response := models.TransactionResponse{
		TransactionID: transactionID,
		MerchantID:    transaction.MerchantID.String(),
		Amount:        transaction.Amount,
		Status:        transaction.Status,
		CreatedAt:     transaction.CreatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    response,
		"message": "payment confirmed successfully",
	})
}
