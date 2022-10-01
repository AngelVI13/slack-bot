package slack

type BaseEvent struct {
	UserName string
	UserId   string
}

func (e BaseEvent) User() string {
	return e.UserName
}
