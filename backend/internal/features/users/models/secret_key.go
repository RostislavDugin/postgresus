package users_models

type SecretKey struct {
	Secret string `gorm:"column:secret"`
}

func (SecretKey) TableName() string {
	return "secret_keys"
}
