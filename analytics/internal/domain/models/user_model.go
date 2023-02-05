package models

// User структура с данными о пользователе
type User struct {
	UserLogin string `json:"login"`
	Email     string `json:"email,omitempty"`
}
