package vo

type ConfigAddVO struct {
	Key         string `json:"key" binding:"required"`
	Value       string `json:"value"`
	Description string `json:"description"`
}
