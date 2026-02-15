package humans

type Human struct {
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	DateOfBirth  string `json:"dateOfBirth"`
	HasAllergies bool   `json:"hasAllergies"`
	Bio          string `json:"bio"`
}
