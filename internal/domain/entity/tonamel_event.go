package entity

type TonamelEvent struct {
	ID          string
	Title       string
	Description string
	Image       string
}

func NewTonamelEvent(
	id string,
	title string,
	description string,
	image string,
) *TonamelEvent {
	return &TonamelEvent{
		ID:          id,
		Title:       title,
		Description: description,
		Image:       image,
	}
}
