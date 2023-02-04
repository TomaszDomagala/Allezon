package api

const GetIDUrl = "/get_id"

type GetIDRequest struct {
	CollectionName string `json:"collection_name"`
	Element        string `json:"element"`
}

type GetIdResponse struct {
	ID int32 `json:"id"`
}
