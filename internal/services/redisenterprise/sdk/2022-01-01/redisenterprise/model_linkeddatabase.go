package redisenterprise

type LinkedDatabase struct {
	Id    *string    `json:"id,omitempty"`
	State *LinkState `json:"state,omitempty"`
}
