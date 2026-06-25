package databases

type CreateReadOnlyUserResponse struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type IsReadOnlyResponse struct {
	IsReadOnly bool     `json:"isReadOnly"`
	Privileges []string `json:"privileges"`
}
