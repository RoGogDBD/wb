// Package models содержит доменные модели приложения.
package models

// Delivery описывает данные доставки.
type Delivery struct {
	Name    string `json:"name" validate:"required"`
	Phone   string `json:"phone" validate:"required,phone_ru"`
	Zip     string `json:"zip" validate:"required,zip_ru"`
	City    string `json:"city" validate:"required"`
	Address string `json:"address" validate:"required"`
	Region  string `json:"region" validate:"required"`
	Email   string `json:"email" validate:"required,email"`
}
