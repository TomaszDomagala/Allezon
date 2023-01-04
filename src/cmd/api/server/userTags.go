package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Shopify/sarama"
	"github.com/gin-gonic/gin"

	"github.com/TomaszDomagala/Allezon/src/pkg/types"
)

type ProductInfo struct {
	ProductId  string `json:"product_id"`
	BrandId    string `json:"brand_id"`
	CategoryId string `json:"category_id"`
	Price      int32  `json:"price"`
}

type UserTagsRequest struct {
	Time        string      `json:"time"`
	Cookie      string      `json:"cookie"`
	Country     string      `json:"country"`
	Device      string      `json:"device"`
	Action      string      `json:"action"`
	Origin      string      `json:"origin"`
	ProductInfo ProductInfo `json:"product_info"`
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

func toTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

func (r *UserTagsRequest) ToUserTag() (types.UserTag, error) {
	t, err := toTime(r.Time)
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

func (s server) sendUserTag(tag types.UserTag) error {
	data, err := json.Marshal(tag)
	if err != nil {
		return err
	}

	msg := &sarama.ProducerMessage{
		Topic: types.UserTagsTopic,
		Value: sarama.ByteEncoder(data),
	}

	_, _, err = s.kafkaProducer.SendMessage(msg)
	return err
}

func (s server) userTagsHandler(c *gin.Context) {
	var req UserTagsRequest
	if err := c.BindJSON(&req); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	userTag, err := req.ToUserTag()
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	if err := s.sendUserTag(userTag); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
	return
}
