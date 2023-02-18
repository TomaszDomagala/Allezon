package types

import (
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/TomaszDomagala/Allezon/src/pkg/types/pbtypes"
)

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

type Aggregate int8

const (
	Sum Aggregate = iota
	Count
)

func (a Aggregate) String() string {
	switch a {
	case Sum:
		return "SUM_PRICE"
	case Count:
		return "COUNT"
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

func MarshalUserTag(tag *UserTag) ([]byte, error) {
	protoTag, err := userTagIntoProto(tag)
	if err != nil {
		return nil, fmt.Errorf("cannot marshall tag: %w", err)
	}
	return proto.Marshal(protoTag)
}

func UnmarshalUserTag(data []byte, tag *UserTag) error {
	var err error

	protoTag := &pbtypes.UserTag{}
	if err = proto.Unmarshal(data, protoTag); err != nil {
		return fmt.Errorf("cannot unmarshall tag: %w", err)
	}
	*tag, err = userTagFromProto(protoTag)
	if err != nil {
		return fmt.Errorf("cannot convert tag: %w", err)
	}
	return nil
}

func userTagIntoProto(tag *UserTag) (*pbtypes.UserTag, error) {
	var action pbtypes.Action

	switch tag.Action {
	case View:
		action = pbtypes.Action_View
	case Buy:
		action = pbtypes.Action_Buy
	default:
		return nil, fmt.Errorf("unknown action: %v", tag.Action)
	}

	return &pbtypes.UserTag{
		Time:        timestamppb.New(tag.Time),
		Cookie:      tag.Cookie,
		Country:     tag.Country,
		Device:      pbtypes.Device(tag.Device),
		Action:      action,
		Origin:      tag.Origin,
		ProductInfo: productInfoIntoProto(tag.ProductInfo),
	}, nil
}

func productInfoIntoProto(info ProductInfo) *pbtypes.ProductInfo {
	return &pbtypes.ProductInfo{
		ProductId:  int32(info.ProductId),
		BrandId:    info.BrandId,
		CategoryId: info.CategoryId,
		Price:      info.Price,
	}
}

func userTagFromProto(tag *pbtypes.UserTag) (UserTag, error) {
	var action Action

	switch tag.Action {
	case pbtypes.Action_View:
		action = View
	case pbtypes.Action_Buy:
		action = Buy
	default:
		return UserTag{}, fmt.Errorf("unknown action: %v", tag.Action)
	}

	return UserTag{
		Time:        tag.Time.AsTime(),
		Cookie:      tag.Cookie,
		Country:     tag.Country,
		Device:      Device(tag.Device),
		Action:      action,
		Origin:      tag.Origin,
		ProductInfo: productInfoFromProto(tag.ProductInfo),
	}, nil
}

func productInfoFromProto(info *pbtypes.ProductInfo) ProductInfo {
	return ProductInfo{
		ProductId:  int(info.ProductId),
		BrandId:    info.BrandId,
		CategoryId: info.CategoryId,
		Price:      info.Price,
	}
}
