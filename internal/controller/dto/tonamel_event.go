package dto

type TonamelEventResponse struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
}

type TonamelEventGetByIdResponse struct {
	TonamelEventResponse
}
