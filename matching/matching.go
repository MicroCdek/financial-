package matching

import (
	"fmt"
	"math"
	"time"

	"enump/enum"
	storage "enump/store"

	"github.com/google/uuid"
)

func ExecuteTrade(
	ob *storage.IndexedStore[uuid.UUID, enum.Order],
	buyOrderID uuid.UUID,
	sellOrderID uuid.UUID,
) (enum.Trade, error) {
	var trade enum.Trade

	// Генерируем UUID сделки
	tradeID, err := uuid.NewV7()
	if err != nil {
		return enum.Trade{}, fmt.Errorf(
			"ошибка генерации UUID: %w", err,
		)
	}

	err = ob.UpdatePair(
		buyOrderID,
		sellOrderID,
		func(buy, sell enum.Order) (enum.Order, enum.Order, error) {
			// Проверяем стороны ордеров
			if buy.Side != enum.SideBuy {
				return buy, sell, fmt.Errorf("первый ордер не является покупкой")
			}

			if sell.Side != enum.SideSell {
				return buy, sell, fmt.Errorf("второй ордер не является продажей")
			}

			// Проверяем активность
			if !buy.IsActive() || !sell.IsActive() {
				return buy, sell, fmt.Errorf("один из ордеров неактивен")
			}

			// Проверяем торговую пару
			if buy.Symbo != sell.Symbo {
				return buy, sell, fmt.Errorf("торговые символы не совпадают")
			}

			// Проверяем цену
			if sell.Price > buy.Price {
				return buy, sell, fmt.Errorf("цена продажи выше цены покупки")
			}

			// Рассчитываем остатки
			buyRemaining := buy.Remaining()
			sellRemaining := sell.Remaining()

			if buyRemaining <= 0 || sellRemaining <= 0 {
				return buy, sell, fmt.Errorf("нет доступного объема для сделки")
			}

			// Определяем количество сделки
			quantity := math.Min(buyRemaining, sellRemaining)

			// Обновляем исполненный объем
			buy.Filled += quantity
			sell.Filled += quantity

			now := time.Now().UTC()

			buy.UpdatedAt = now
			sell.UpdatedAt = now

			// Обновляем статус покупки
			if buy.Remaining() <= 0 {
				buy.Status = enum.OrderStatusFilled
			} else {
				buy.Status = enum.OrderStatusPartiallyFilled
			}

			// Обновляем статус продажи
			if sell.Remaining() <= 0 {
				sell.Status = enum.OrderStatusFilled
			} else {
				sell.Status = enum.OrderStatusPartiallyFilled
			}

			// Формируем сделку
			trade = enum.Trade{
				ID:         tradeID,
				Symbo:      buy.Symbo,
				BuyOrderID: buy.ID,
				SelOrderID: sell.ID,
				BuyerID:    buy.TraderID,
				SellerID:   sell.TraderID,
				Price:      sell.Price,
				Quantity:   quantity,
				ExecutedAt: now,
			}

			return buy, sell, nil
		},
	)
	if err != nil {
		return enum.Trade{}, err
	}

	return trade, nil
}
