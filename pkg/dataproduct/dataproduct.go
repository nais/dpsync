package dataproduct

type Dataproduct struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Team        string            `json:"team"`
	Datastore   map[string]string `json:"datastore"`
}
