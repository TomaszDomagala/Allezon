package api

const GetIDUrl = "/get_id"

type GetIDRequest struct {
	CollectionName string `json:"collection_name"`
	Element        string `json:"element"`
	CreateMissing  bool   `json:"create_missing"`
}

type GetIdResponse struct {
	ID int32 `json:"id"`
}
