package vo

type ConfigUpdateVO struct {
	ID          int64  `json:"id,string" binding:"required"`
	Key         string `json:"key" binding:"required"`
	Value       string `json:"value"`
	Description string `json:"description"`
}
