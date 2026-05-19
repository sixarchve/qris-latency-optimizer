package redis

import (
	"encoding/json"
	"fmt"
	"qris-latency-optimizer/domain/entity"
)

func transactionCacheKey(id string) string {
	return fmt.Sprintf("transaction:%s", id)
}

func GetTransaction(id string) (*entity.Transaction, bool) {
	if !RedisAvailable || id == "" {
		return nil, false
	}
	cachedData, err := Get(transactionCacheKey(id))
	if err != nil || cachedData == "" {
		return nil, false
	}
	var tx entity.Transaction
	if err := json.Unmarshal([]byte(cachedData), &tx); err != nil {
		_ = Delete(transactionCacheKey(id))
		return nil, false
	}
	return &tx, true
}

func CacheTransaction(tx entity.Transaction) {
	if !RedisAvailable || tx.ID.String() == "" {
		return
	}
	if data, err := json.Marshal(tx); err == nil {
		_ = Set(transactionCacheKey(tx.ID.String()), string(data), TTLTransaction)
	}
}

func DeleteTransaction(id string) {
	_ = Delete(transactionCacheKey(id))
}
