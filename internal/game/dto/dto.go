package dto

type UpsertGameRequest struct {
	Code        string `json:"code" binding:"required,max=80"`
	Name        string `json:"name" binding:"required,max=160"`
	PackageName string `json:"package_name" binding:"max=200"`
	Timezone    string `json:"timezone" binding:"required"`
	Currency    string `json:"currency" binding:"required,len=3"`
	Status      string `json:"status" binding:"required,oneof=ACTIVE INACTIVE"`
}
