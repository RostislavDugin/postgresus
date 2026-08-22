package databases

type CreateReadOnlyUserResponse struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ShouldSuggestReadOnlyUserResponse struct {
	ShouldSuggestReadOnlyUser bool     `json:"shouldSuggestReadOnlyUser"`
	Privileges                []string `json:"privileges"`
}
