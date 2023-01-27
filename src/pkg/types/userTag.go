package types

import "time"

type Device int8

const (
	Pc Device = iota
	Mobile
	Tv
)

func (d Device) String() string {
	switch d {
	case Pc:
		return "PC"
	case Mobile:
		return "MOBILE"
	case Tv:
		return "TV"
	default:
		return "Unknown"
	}
}

type Action int8

const (
	View Action = iota
	Buy
)

func (a Action) String() string {
	switch a {
	case View:
		return "VIEW"
	case Buy:
		return "BUY"
	default:
		return "Unknown"
	}
}

type ProductInfo struct {
	ProductId  int    `json:"product_id"`
	BrandId    string `json:"brand_id"`
	CategoryId string `json:"category_id"`
	Price      uint32 `json:"price"`
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
