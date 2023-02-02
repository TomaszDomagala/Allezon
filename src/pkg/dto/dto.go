package dto

import (
	"fmt"
	"github.com/TomaszDomagala/Allezon/src/pkg/types"
	"time"
)

const UserTagTimeLayout = "2006-01-02T15:04:05.999Z"

// UserTagDTO is a data transfer object for types.UserTag.
type UserTagDTO struct {
	Time        string      `json:"time"`
	Cookie      string      `json:"cookie"`
	Country     string      `json:"country"`
	Device      string      `json:"device"`
	Action      string      `json:"action"`
	Origin      string      `json:"origin"`
	ProductInfo ProductInfo `json:"product_info"`
}

// ProductInfo is a data transfer object for types.ProductInfo.
type ProductInfo struct {
	ProductId  int    `json:"product_id"`
	BrandId    string `json:"brand_id"`
	CategoryId string `json:"category_id"`
	Price      uint32 `json:"price"`
}

// UserProfileDTO is a data transfer object for user profile.
type UserProfileDTO struct {
	Cookie string       `json:"cookie"`
	Views  []UserTagDTO `json:"views"`
	Buys   []UserTagDTO `json:"buys"`
}

// FromUserTagDTO converts UserTagDTO to types.UserTag.
func FromUserTagDTO(dto UserTagDTO) (types.UserTag, error) {
	t, err := time.Parse(UserTagTimeLayout, dto.Time)
	if err != nil {
		return types.UserTag{}, err
	}
	device, err := toDevice(dto.Device)
	if err != nil {
		return types.UserTag{}, err
	}
	action, err := toAction(dto.Action)
	if err != nil {
		return types.UserTag{}, err
	}

	return types.UserTag{
		Time:    t,
		Cookie:  dto.Cookie,
		Country: dto.Country,
		Device:  device,
		Action:  action,
		Origin:  dto.Origin,
		ProductInfo: types.ProductInfo{
			ProductId:  dto.ProductInfo.ProductId,
			BrandId:    dto.ProductInfo.BrandId,
			CategoryId: dto.ProductInfo.CategoryId,
			Price:      dto.ProductInfo.Price,
		},
	}, nil
}

// IntoUserTagDTO converts types.UserTag to UserTagDTO.
func IntoUserTagDTO(tag types.UserTag) UserTagDTO {
	return UserTagDTO{
		Time:    tag.Time.Format(UserTagTimeLayout),
		Cookie:  tag.Cookie,
		Country: tag.Country,
		Device:  tag.Device.String(),
		Action:  tag.Action.String(),
		Origin:  tag.Origin,
		ProductInfo: ProductInfo{
			ProductId:  tag.ProductInfo.ProductId,
			BrandId:    tag.ProductInfo.BrandId,
			CategoryId: tag.ProductInfo.CategoryId,
			Price:      tag.ProductInfo.Price,
		},
	}
}

func toDevice(s string) (types.Device, error) {
	switch s {
	case "PC":
		return types.Pc, nil
	case "MOBILE":
		return types.Mobile, nil
	case "TV":
		return types.Tv, nil
	default:
		return 0, fmt.Errorf("can't convert to device: %s", s)
	}
}

func toAction(s string) (types.Action, error) {
	switch s {
	case "VIEW":
		return types.View, nil
	case "BUY":
		return types.Buy, nil
	default:
		return 0, fmt.Errorf("can't convert to action: %s", s)
	}
}
