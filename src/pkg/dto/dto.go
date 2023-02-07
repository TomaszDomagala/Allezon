package dto

import (
	"fmt"
	"time"

	"github.com/TomaszDomagala/Allezon/src/pkg/types"
)

const UserTagTimeLayout = "2006-01-02T15:04:05.999Z"
const TimeRangeMilliPrecisionLayout = "2006-01-02T15:04:05.999"
const TimeRangeSecPrecisionLayout = "2006-01-02T15:04:05"

// UserTagDTO is a data transfer object for types.UserTag.
type UserTagDTO struct {
	Time        string      `json:"time" binding:"required"`
	Cookie      string      `json:"cookie" binding:"required"`
	Country     string      `json:"country" binding:"required"`
	Device      string      `json:"device" binding:"required,oneof=PC TV MOBILE"`
	Action      string      `json:"action" binding:"required,oneof=VIEW BUY"`
	Origin      string      `json:"origin" binding:"required"`
	ProductInfo ProductInfo `json:"product_info" binding:"required"`
}

// ProductInfo is a data transfer object for types.ProductInfo.
type ProductInfo struct {
	ProductID  int    `json:"product_id" binding:"required"`
	BrandID    string `json:"brand_id" binding:"required"`
	CategoryID string `json:"category_id" binding:"required"`
	Price      uint32 `json:"price" binding:"required"`
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
			ProductId:  dto.ProductInfo.ProductID,
			BrandId:    dto.ProductInfo.BrandID,
			CategoryId: dto.ProductInfo.CategoryID,
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
			ProductID:  tag.ProductInfo.ProductId,
			BrandID:    tag.ProductInfo.BrandId,
			CategoryID: tag.ProductInfo.CategoryId,
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

func ToAction(s string) (types.Action, error) {
	switch s {
	case "VIEW":
		return types.View, nil
	case "BUY":
		return types.Buy, nil
	default:
		return 0, fmt.Errorf("can't convert to action: %s", s)
	}
}
