package handler

import (
	"net/http"
	"strconv"

	"qris-latency-optimizer/usecase/service"

	"github.com/gin-gonic/gin"
	"github.com/skip2/go-qrcode"
)

func GenerateQRISHandler(c *gin.Context) {
	amountStr := c.Query("amount")

	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid amount",
		})
		return
	}

	// generate payload QRIS
	qrPayload := service.GenerateQRIS(amount)

	// generate QR image (PNG in memory)
	png, err := qrcode.Encode(qrPayload, qrcode.Medium, 256)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate QR",
		})
		return
	}

	// return image
	c.Data(http.StatusOK, "image/png", png)
}