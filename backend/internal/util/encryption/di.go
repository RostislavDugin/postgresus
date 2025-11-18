package encryption

import users_repositories "postgresus-backend/internal/features/users/repositories"

var fieldEncryptor = &SecretKeyFieldEncryptor{
	users_repositories.GetSecretKeyRepository(),
}

func GetFieldEncryptor() FieldEncryptor {
	return fieldEncryptor
}
