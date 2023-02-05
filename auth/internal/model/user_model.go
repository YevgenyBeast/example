package model

// User структура с данными о пользователе
type User struct {
	ID       string `json:"id,omitempty" bson:"_id" example:"54084cbe-2b1c-4829-9720-8a36202f79ce"`
	Username string `json:"login" bson:"username" example:"TestUser"`
	Password string `json:"password,omitempty" bson:"passwordhash" example:"DerParol"`
	Email    string `json:"email,omitempty" bson:"email" example:"test@mail.com"`
}
