package domain

type Config struct {
	DisplayName string
	OwnerToken  string
}

func NewConfig(displayName, ownerToken string) Config {
	return Config{
		DisplayName: displayName,
		OwnerToken:  ownerToken,
	}
}
