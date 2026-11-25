package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"enump/enum"
	storage "enump/store"

	"github.com/google/uuid"
)

func main() {
	// Настройка логирования.
	options := slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	handler := slog.NewJSONHandler(os.Stdout, &options)
	logger := slog.New(handler)

	// Создаём трейдеров.
	buyerID := uuid.New()
	sellerID := uuid.New()

	// Хранилище балансов трейдеров.
	traderStore := storage.NewStore[uuid.UUID, float64](10)

	traderStore.Put(buyerID, 50000.0004)
	traderStore.Put(sellerID, 50000.0004)

	logger.Info(
		"Новые трейдеры добавлены в хранилище",
		slog.String("BuyerID", buyerID.String()),
		slog.String("SellerID", sellerID.String()),
	)

	// Проверка получения баланса покупателя.
	if balance, ok := traderStore.Get(buyerID); ok {
		logger.Info(
			"Баланс покупателя",
			slog.String("UUID", buyerID.String()),
			slog.Float64("Balance", balance),
		)
	} else {
		logger.Error("Покупатель не найден")
		return
	}

	// Шардированное хранилище ордеров.
	orderBook := storage.NewShardedOrderBook(16, 100)

	symbo := "BTC-USDT"

	uuidOrder1 := uuid.New()
	uuidOrder2 := uuid.New()

	now := time.Now().UTC()

	// Создание BUY ордера.
	orderBuy := enum.Order{
		ID:        uuidOrder1,
		TraderID:  buyerID,
		Side:      enum.SideBuy,
		Type:      enum.OrderTypeLimit,
		Symbo:     symbo,
		Price:     1000.00,
		Quantity:  2.00,
		Filled:    1.00,
		Status:    enum.OrderStatusPartiallyFilled,
		CreatedAt: now,
		UpdatedAt: now,
	}

	logger.Info("Создание BUY order", slog.Any("order", &orderBuy))

	// Создание SELL ордера.
	orderSell := enum.Order{
		ID:        uuidOrder2,
		TraderID:  sellerID,
		Side:      enum.SideSell,
		Type:      enum.OrderTypeLimit,
		Symbo:     symbo,
		Price:     1000.00,
		Quantity:  2.00,
		Filled:    0.00,
		Status:    enum.OrderStatusNew,
		CreatedAt: now,
		UpdatedAt: now,
	}

	logger.Info("Создание SELL order", slog.Any("order", &orderSell))

	// Добавление ордеров в OrderBook.
	if err := orderBook.Put(orderBuy); err != nil {
		logger.Error(
			"Ошибка добавления BUY ордера",
			slog.String("error", err.Error()),
		)
		return
	}

	if err := orderBook.Put(orderSell); err != nil {
		logger.Error(
			"Ошибка добавления SELL ордера",
			slog.String("error", err.Error()),
		)
		return
	}

	logger.Info("Ордера успешно добавлены в OrderBook")

	// Проверка получения ордера.
	if order, ok := orderBook.Get(symbo, uuidOrder1); ok {
		logger.Info(
			"BUY ордер найден",
			slog.Any("order", order),
		)
	} else {
		logger.Error("BUY ордер не найден")
		return
	}

	// Исполнение сделки.
	var trade enum.Trade

	err := orderBook.UpdatePair(
		symbo,
		uuidOrder1,
		uuidOrder2,
		func(buy, sell enum.Order) (enum.Order, enum.Order, error) {
			// Проверяем стороны ордеров.
			if buy.Side != enum.SideBuy || sell.Side != enum.SideSell {
				return buy, sell, fmt.Errorf(
					"неправильные стороны ордеров",
				)
			}

			// Проверяем торговую пару.
			if buy.Symbo != sell.Symbo {
				return buy, sell, fmt.Errorf(
					"ордера принадлежат разным торговым парам",
				)
			}

			// Проверяем возможность исполнения.
			if buy.Price < sell.Price {
				return buy, sell, fmt.Errorf(
					"цена покупки ниже цены продажи",
				)
			}

			// Проверяем активность ордеров.
			if !buy.IsActive() || !sell.IsActive() {
				return buy, sell, fmt.Errorf(
					"один из ордеров неактивен",
				)
			}

			// Проверяем оставшееся количество.
			buyRemaining := buy.Remaining()
			sellRemaining := sell.Remaining()

			if buyRemaining <= 0 || sellRemaining <= 0 {
				return buy, sell, fmt.Errorf(
					"нет доступного количества для сделки",
				)
			}

			// Определяем количество сделки.
			quantity := buyRemaining

			if sellRemaining < quantity {
				quantity = sellRemaining
			}

			// Обновляем BUY и SELL ордера.
			buy.Fill(quantity)
			sell.Fill(quantity)

			// Формируем информацию о сделке.
			trade = enum.Trade{
				ID:         uuid.New(),
				Symbo:      symbo,
				BuyOrderID: buy.ID,
				SelOrderID: sell.ID,
				BuyerID:    buy.TraderID,
				SellerID:   sell.TraderID,
				Price:      sell.Price,
				Quantity:   quantity,
				ExecutedAt: time.Now().UTC(),
			}

			return buy, sell, nil
		},
	)
	if err != nil {
		logger.Error(
			"Ошибка исполнения сделки",
			slog.String("error", err.Error()),
		)
		return
	}

	// Логирование исполненной сделки.
	logger.Info(
		"Сделка успешно исполнена",
		slog.String("TradeID", trade.ID.String()),
		slog.String("Symbo", trade.Symbo),
		slog.String("BuyerID", trade.BuyerID.String()),
		slog.String("SellerID", trade.SellerID.String()),
		slog.Float64("Price", trade.Price),
		slog.Float64("Quantity", trade.Quantity),
		slog.String("ExecutedAt", trade.ExecutedAt.String()),
	)

	// Проверяем BUY после исполнения.
	if buy, ok := orderBook.Get(symbo, uuidOrder1); ok {
		logger.Info(
			"BUY после сделки",
			slog.Any("order", buy),
		)
	}

	// Проверяем SELL после исполнения.
	if sell, ok := orderBook.Get(symbo, uuidOrder2); ok {
		logger.Info(
			"SELL после сделки",
			slog.Any("order", sell),
		)
	}

	fmt.Scanln()
}
