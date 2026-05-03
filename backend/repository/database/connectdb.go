package database

import (
	"qris-latency-optimizer/models"

  	"gorm.io/driver/postgres"
  	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDB() {
	var err error
	
	DB, err = gorm.Open(postgres.Open(LoadDatabaseConfig()), &gorm.Config{})

	if err != nil {
		panic(err)
	}

	var c models.Merchant
	var d models.Transaction

	DB.AutoMigrate(&c)
	DB.AutoMigrate(&d)

}