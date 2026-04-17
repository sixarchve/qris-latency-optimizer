package handler

import (
	"net/http"
	"strconv"

	"qris-latency-optimizer/usecase/service"

	"github.com/gin-gonic/gin"
)

func GenerateQRIS(c *gin.Context) {
	amountStr := c.Query("amount")

	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid amount",
		})
		return
	}

	qr := service.GenerateQRIS(amount)

	c.JSON(http.StatusOK, gin.H{
		"qris_payload": qr,
	})
}