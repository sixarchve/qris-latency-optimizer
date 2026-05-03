package service

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



// GetTransactionStatus - endpoint untuk check status transaksi
func GetTransactionStatus(c *gin.Context) {
	transactionID := c.Param("id")

	// Validasi UUID
	if _, err := uuid.Parse(transactionID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid transaction id",
		})
		return
	}

	cacheKey := fmt.Sprintf("transaction:%s", transactionID)

	// Cek di Redis dulu (cache)
	cachedData, err := redis.Get(cacheKey)

	// Debug: Print untuk lihat status cache
	if err != nil {
		fmt.Printf("Redis Get Error: %v\n", err)
	}
	fmt.Printf("Cache Key: %s | Data Exists: %v | Error: %v\n", cacheKey, cachedData != "", err)

	if err == nil && cachedData != "" {
		fmt.Println("✓ Cache HIT - returning cached data")
		// Cache hit!
		var transaction models.Transaction
		json.Unmarshal([]byte(cachedData), &transaction)

		response := models.TransactionResponse{
			TransactionID: transactionID,
			MerchantID:    transaction.MerchantID.String(),
			Amount:        transaction.Amount,
			Status:        transaction.Status,
			CreatedAt:     transaction.CreatedAt,
			CachedFrom:    true,
		}

		c.JSON(http.StatusOK, gin.H{
			"data":    response,
			"message": "transaction data (from cache)",
		})
		return
	}

	fmt.Println("✗ Cache MISS - querying database")

	// Cache miss - query database
	var transaction models.Transaction
	if err := database.DB.First(&transaction, "id = ?", transactionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "transaction not found",
		})
		return
	}

	// Simpan ke Redis untuk next request
	transactionJSON, _ := json.Marshal(transaction)
	saveErr := redis.Set(cacheKey, string(transactionJSON), 10*time.Minute)
	if saveErr != nil {
		fmt.Printf("Failed to save to Redis: %v\n", saveErr)
	} else {
		fmt.Println("✓ Data saved to Redis cache")
	}

	response := models.TransactionResponse{
		TransactionID: transactionID,
		MerchantID:    transaction.MerchantID.String(),
		Amount:        transaction.Amount,
		Status:        transaction.Status,
		CreatedAt:     transaction.CreatedAt,
		CachedFrom:    false,
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    response,
		"message": "transaction data (from database)",
	})
}

