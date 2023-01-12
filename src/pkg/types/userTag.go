package types

import "time"

type Device int8

const (
	Pc Device = iota
	Mobile
	Tv
)

type Action int8

const (
	View Action = iota
	Buy
)

type ProductInfo struct {
	ProductId  int    `json:"product_id"`
	BrandId    string `json:"brand_id"`
	CategoryId string `json:"category_id"`
	Price      int32  `json:"price"`
}

type UserTag struct {
	Time        time.Time   `json:"time"`
	Cookie      string      `json:"cookie"`
	Country     string      `json:"country"`
	Device      Device      `json:"device"`
	Action      Action      `json:"action"`
	Origin      string      `json:"origin"`
	ProductInfo ProductInfo `json:"product_info"`
}
