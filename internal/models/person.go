package models

type Person struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Surname     string  `json:"surname"`
	Patronymic  *string `json:"patronymic,omitempty"`
	Age         *int    `json:"age,omitempty"`
	Gender      *string `json:"gender,omitempty" enums:"male,female,other"`
	Nationality *string `json:"nationality,omitempty"`
}

type PersonRequest struct {
	Name       string  `json:"name" binding:"required"`
	Surname    string  `json:"surname" binding:"required"`
	Patronymic *string `json:"patronymic,omitempty"`
}

type PersonPatch struct {
	Name        *string `json:"name,omitempty"`
	Surname     *string `json:"surname,omitempty"`
	Patronymic  *string `json:"patronymic,omitempty"`
	Age         *int    `json:"age,omitempty"`
	Gender      *string `json:"gender,omitempty" enums:"male,female,other"`
	Nationality *string `json:"nationality,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
