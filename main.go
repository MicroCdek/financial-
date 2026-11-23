package main

import (
	"enump/enum"
	"enump/modifiers"
	storage "enump/store"
	"fmt"
	"log/slog"
	"os"
	"time"
	"enump/matching"
	"github.com/google/uuid"
)

func main() {
	options := slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewJSONHandler(os.Stdout, &options)
	logger := slog.New(handler)

	// currencies := storage.NewStorage[string](5)
	// currencies.Set("BTC-RUB")
	// currencies.Set("ETH-RUB")
	// currencies.Set("USD-RUB")
	// currencies.Set("USD-RUB")
	//
	// lengthOfCurrencies := currencies.Lenght()
	// logger.Info("Работа с валютами:", slog.Int("Количество уникальных комбинаций", lengthOfCurrencies))
	//
	// hasOfCurrenices := currencies.Has("USD-RUB")
	// logger.Info("Работа с валютами: ", slog.Bool("Наличие USD-RUB", hasOfCurrenices))
	//
	// currencies.Remove("USD-RUB")
	// lengthOfRemoveCurren := currencies.Lenght()
	// logger.Info("Работа с валютами:", slog.Int("Количество уникальных комбинций после удаления", lengthOfRemoveCurren))
	//
	traderStore := storage.NewStore[uuid.UUID, float64](10)
	traderID := uuid.New()

	traderStore.Put(traderID, 50000.0004)
	logger.Info("Новый трейдер добавлен в хранилище:", slog.String("UUID", traderID.String()), slog.Float64("Начальный баланс", 50000.0004))

	if balance, ok := traderStore.Get(traderID); ok {
		logger.Info("поиск в хранилище:", slog.String("Найден трейдер с UUID", traderID.String()), slog.Float64("его текущий баланс", balance))
	} else {
		logger.Error("элемнт не найден в хранилище:", slog.String("проверьте UUID", traderID.String()))
	}

	orderBook := storage.NewIndexedStore[uuid.UUID, enum.Order](100)
	
	uuidOrder1 := uuid.New()
	uuidOrder2 := uuid.New()

	orderBuy := enum.Order{
		ID:        uuidOrder1,
		TraderID:  uuid.New(),
		Side:      enum.SideBuy,
		Type:      enum.OrderTypeLimit,
		Symbo:     "USD",
		Price:     1000.00,
		Quantity:  2.00,
		Filled:    1.00,
		Status:    enum.OrderStatusNew,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	enum.LogOrderState(&orderBuy, logger)
	orderSell := enum.Order{
		ID:        uuidOrder2,
		TraderID:  uuid.New(),
		Side:      enum.SideBuy,
		Type:      enum.OrderTypeLimit,
		Symbo:     "USD",
		Price:     1000.00,
		Quantity:  2.00,
		Filled:    1.00,
		Status:    enum.OrderStatusNew,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	enum.LogOrderState(&orderSell, logger)

	orderBook.Put(uuidOrder1, orderBuy)
	orderBook.Put(uuidOrder2, orderSell)

	// logger.Info("Зарегистрирована сделка",
	// 	slog.String("Type", orderBuy.Type.String()),
	// 	slog.Float64("Value:", orderBuy.Price),
	// 	slog.Float64("Quantity", orderBuy.Quantity),
	// 	slog.String("Symbo", orderBuy.Symbo),
	// 	slog.String("TraderID", orderBuy.TraderID.String()),
	// )
	//
	// orderBook.Delete(uuidOrder1)
	// logger.Info("Отменена сделка",
	// 	slog.String("Type", orderBuy.Type.String()),
	// 	slog.Float64("Value:", orderBuy.Price),
	// 	slog.Float64("Quantity", orderBuy.Quantity),
	// 	slog.String("Symbo", orderBuy.Symbo),
	// 	slog.String("TraderID", orderBuy.TraderID.String()),
	// )

orderBook1 := storage.NewStore[uuid.UUID, enum.Order](10)

orderBook1.Update(uuidOrder1, modifiers.ModifierForOrderBuy(orderSell.Quantity), logger)
orderBook1.Update(uuidOrder2, modifiers.ModifierForOrderBuy(orderBuy.Quantity), logger)
 // Новый вывод
	orderSell.Side = enum.SideSell

	orderBook.Put(uuidOrder1, orderBuy)
	orderBook.Put(uuidOrder2, orderSell)


	// Выполняем сделка
	trade, err := matching.ExecuteTrade(orderBook, uuidOrder1, uuidOrder2)
	if err != nil {
		logger.Error("Ошибка исполнения сделки", slog.String("error", err.Error()),)
	} else{
		logger.Info("Сделка исполнена", slog.String("TraderID", trade.ID.String()),
		slog.Float64("Quantity", trade.Quantity),
		slog.Float64("Price", trade.Price),
		)
	}
	fmt.Scanln()
}
