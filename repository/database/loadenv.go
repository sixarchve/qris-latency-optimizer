package database

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	err := godotenv.Load(".env")
	env := os.Getenv("ENV")
	if err != nil && env == "" {
		log.Fatalf("error load env : %v", err)
	}
}