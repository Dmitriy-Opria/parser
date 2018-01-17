package model

import "time"

type (
	Coin struct {
		Name       string    `json:"symbol"`
		Price      string    `json:"price"`
		InsertTime time.Time `json:"insert_time"`
	}
)
