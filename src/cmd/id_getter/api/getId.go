package api

type GetIdRequest struct {
	CollectionName string `json:"collection_name"`
	Element        string `json:"element"`
}

type GetIdResponse struct {
	Id int32 `json:"id"`
}
