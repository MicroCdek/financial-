package enum

import (
	"math"
	"time"
	"github.com/google/uuid"
)

type Side uint8

const (
	SideBuy Side = iota + 1
	SideSell
)

func (side Side) String() string {
	switch side {
	case SideBuy:
		return "Покупка"
	case SideSell:
		return "Продажа"
	default:
		return "Операция не определена"
	}
}

type OrderType int8

const (
	OrderTypeLimit OrderType = iota + 1
	OrderTypeMarket
)

func (t OrderType) String() string {
	switch t {
	case OrderTypeLimit:
		return "Заявка на фиксированную цену"
	case OrderTypeMarket:
		return "Заявка по рыночной стоимости"
	default:
		return "Операция не Тип заявки не определен"
	}
}

type OrderStatus int8

const (
	OrderStatusNew             OrderStatus = iota + 1 // Зарегистрирована на бирже
	OrderStatusPartiallyFilled                        // Частично выполнена
	OrderStatusFilled                                 // Выполнена
	OrderStatusCancelled                              // Отменена трейдером
	OrderStatusRejected                               // Отменена биржей (несоблюдение правил или недостаточно средств для осуществления сделки)
)

func (status OrderStatus) String() string {
	switch status {
	case OrderStatusNew:
		return "Заявка зарегистрирована"
	case OrderStatusPartiallyFilled:
		return "Заявка частично выполнена"
	case OrderStatusFilled:
		return "Заявка выполнена"
	case OrderStatusCancelled:
		return "Трейдер отменил заявку"
	case OrderStatusRejected:
		return "Заявка заморожена биржей"
	default:
		return "Статус не определяется"
	}
}

type Trade struct {
	ID         uuid.UUID
	Symbol     string
	BuyOrderID uuid.UUID
	SelOrderID uuid.UUID
	BuyerID    uuid.UUID
	SellerID   uuid.UUID
	Price      float64
	Quantity   float64
	ExecutedAt time.Time
}

type Order struct {
	ID        uuid.UUID
	TraderID  uuid.UUID
	Side      Side
	Type      OrderType
	Symbo     string  // AAPL
	Price     float64 // за единицу товара
	Quantity  float64 // желаемый объем при совершении сделки
	Filled    float64 // частично исполненный объем по заказу
	Status    OrderStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (order *Order) Remaining() float64 {
	return order.Quantity - order.Filled
}

func (order *Order) IsActive() bool {
	return order.Status == OrderStatusNew || order.Status == OrderStatusPartiallyFilled
}

// position1.price = 0.1
// position2.price = 0.2
// return position1.price + position2.price == 0.300000004
type Position struct {
	TraderID uuid.UUID
	Symbol   string
	Quantity float64
	AvgPrice float64
	Realized float64
}

// избавиление от багов с плавающей запятой в float
func RoundPrice(price float64) float64 {
	return math.Round(price*100) / 100
}
