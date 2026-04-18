package service

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GenerateDynamic(c *gin.Context) {
	amountStr := c.Query("amount")

	amount, err := strconv.Atoi(amountStr)
	if err != nil || amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid amount",
		})
		return
	}

	qr := GenerateQRIS(amount)

	c.JSON(http.StatusOK, gin.H{
		"qris_payload": qr,
	})
}