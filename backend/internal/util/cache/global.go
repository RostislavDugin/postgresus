package cache

var defaultStore Store = NewMemoryStore(DefaultPayloadBudgetBytes)

func GetStore() Store {
	return defaultStore
}
