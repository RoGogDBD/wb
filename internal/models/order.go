package models

import "time"

// Структура заказа
type Order struct {
	OrderUID          string    `json:"order_uid" validate:"required,uuid"`
	TrackNumber       string    `json:"track_number" validate:"required"`
	Entry             string    `json:"entry" validate:"required"`
	Delivery          Delivery  `json:"delivery" validate:"required"`
	Payment           Payment   `json:"payment" validate:"required"`
	Items             []Item    `json:"items" validate:"required,min=1,dive"`
	Locale            string    `json:"locale" validate:"required"`
	InternalSignature string    `json:"internal_signature"`
	CustomerID        string    `json:"customer_id" validate:"required"`
	DeliveryService   string    `json:"delivery_service" validate:"required"`
	ShardKey          string    `json:"shardkey" validate:"required"`
	SmID              int       `json:"sm_id" validate:"gte=0"`
	DateCreated       time.Time `json:"date_created" validate:"required"`
	OofShard          string    `json:"oof_shard" validate:"required"`
}
