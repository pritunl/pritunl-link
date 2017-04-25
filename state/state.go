package state

type State struct {
	Id     string
	Secret string
	Hosts  []string
	Links  []*Link
}

type Link struct {
	PreSharedKey string
	Right        string
	LeftSubnets  []string
	RightSubnets []string
}
