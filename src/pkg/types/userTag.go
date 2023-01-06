package types

import "time"

const UserTagsTopic string = "user-tags"

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
	ProductId  int
	BrandId    string
	CategoryId string
	Price      int32
}

type UserTag struct {
	Time        time.Time
	Cookie      string
	Country     string
	Device      Device
	Action      Action
	Origin      string
	ProductInfo ProductInfo
}
