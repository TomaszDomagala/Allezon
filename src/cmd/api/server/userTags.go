package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"

	"github.com/TomaszDomagala/Allezon/src/pkg/types"
)

type UserTagsJson struct {
	Time        string      `json:"time"`
	Cookie      string      `json:"cookie"`
	Country     string      `json:"country"`
	Device      string      `json:"device"`
	Action      string      `json:"action"`
	Origin      string      `json:"origin"`
	ProductInfo ProductInfo `json:"product_info"`
}

type ProductInfo struct {
	ProductId  int    `json:"product_id"`
	BrandId    string `json:"brand_id"`
	CategoryId string `json:"category_id"`
	Price      int32  `json:"price"`
}

type UserTagsRequest = UserTagsJson

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

const userTagTimeLayout = "2022-03-22T12:15:00.000Z"

func (r *UserTagsRequest) ToUserTag() (types.UserTag, error) {
	t, err := time.Parse(userTagTimeLayout, r.Time)
	if err != nil {
		return types.UserTag{}, err
	}
	device, err := toDevice(r.Device)
	if err != nil {
		return types.UserTag{}, err
	}
	action, err := toAction(r.Action)
	if err != nil {
		return types.UserTag{}, err
	}

	return types.UserTag{
		Time:    t,
		Cookie:  r.Cookie,
		Country: r.Country,
		Device:  device,
		Action:  action,
		Origin:  r.Origin,
		ProductInfo: types.ProductInfo{
			ProductId:  r.ProductInfo.ProductId,
			BrandId:    r.ProductInfo.BrandId,
			CategoryId: r.ProductInfo.CategoryId,
			Price:      r.ProductInfo.Price,
		},
	}, nil
}

func (s server) userTagsHandler(c *gin.Context) {
	var req UserTagsRequest

	body, err := c.GetRawData()
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		s.logger.Error("can't unmarshal request: %s", zap.Error(err), zap.ByteString("body", body))
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	userTag, err := req.ToUserTag()
	if err != nil {
		s.logger.Error("can't convert request to user tag: %s", zap.Error(err), zap.ByteString("body", body))
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	if err := s.producer.Send(userTag); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
	return
}
