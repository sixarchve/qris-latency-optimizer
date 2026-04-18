package service

import (
	"fmt"
	"strconv"
)

func tlv(tag string, value string) string {
	length := fmt.Sprintf("%02d", len(value))
	return tag + length + value
}

func crc16(input string) string {
	var crc uint16 = 0xFFFF
	for i := 0; i < len(input); i++ {
		crc ^= uint16(input[i]) << 8
		for j := 0; j < 8; j++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return fmt.Sprintf("%04X", crc)
}

func GenerateQRIS(amount int) string {
	payload := ""

	// payload format
	payload += tlv("00", "01")

	// dynamic QR
	payload += tlv("01", "12")

	// merchant info (dummy)
	merchant := ""
	merchant += tlv("00", "ID.CO.QRIS.WWW")
	merchant += tlv("01", "1234567890")
	payload += tlv("26", merchant)

	// MCC
	payload += tlv("52", "5411")

	// currency IDR
	payload += tlv("53", "360")

	// amount
	payload += tlv("54", strconv.Itoa(amount))

	// country
	payload += tlv("58", "ID")

	// merchant name
	payload += tlv("59", "AETHER STORE")

	// city
	payload += tlv("60", "MALANG")

	// CRC placeholder
	payload += "6304"

	crc := crc16(payload)
	payload += crc

	return payload
}