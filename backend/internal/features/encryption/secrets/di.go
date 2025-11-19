package secrets

var secretKeyService = &SecretKeyService{
	nil,
}

func GetSecretKeyService() *SecretKeyService {
	return secretKeyService
}
