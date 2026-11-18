package modifiers

import (
	"enump/enum"
)

// modifierForOrderBuy возвращает модификатор для обновления ордера на покупку.
func ModifierForOrderBuy(sellerQuantity float64) func(enum.Order) enum.Order {
	return func(order enum.Order) enum.Order {
		// Обновляем исполненное количество.
		order.Filled += sellerQuantity

		// Обновляем статус ордера.
		if order.Filled >= order.Quantity {
			order.Status = enum.OrderStatusFilled
		} else {
			order.Status = enum.OrderStatusPartiallyFilled
		}

		// Возвращаем обновлённый ордер.
		return order
	}
}
