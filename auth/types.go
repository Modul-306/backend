package auth

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type SignUpCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}
