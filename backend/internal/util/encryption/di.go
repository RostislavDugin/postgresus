package encryption

import "databasus-backend/internal/features/encryption/secrets"

var fieldEncryptor = &SecretKeyFieldEncryptor{
	secrets.GetSecretKeyService(),
}

func GetFieldEncryptor() FieldEncryptor {
	return fieldEncryptor
}
