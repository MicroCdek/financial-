package storage

import (
	"fmt"
	"enump/enum"
	"log/slog"
	"sync"
)

// sync.Mutex полностью блокирует доступ и для чтения и для записи
// RWMutex позволяет многим потокам читать данные одновременно, но запись осуществляется только одним человеком
type Store[K comparable, V any] struct {
	mutex sync.RWMutex
	items map[K]V
}

func NewStore[K comparable, V any](cap int) *Store[K, V] {
	return &Store[K, V]{
		items: make(map[K]V, cap),
	}
}

// доюавление под Lock()
func (store *Store[K, V]) Put(key K, value V) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	store.items[key] = value
}

// чтение под RLock()
func (store *Store[K, V]) Get(key K) (V, bool) {
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	value, ok := store.items[key]
	return value, ok
}

// удаление под Lock()
func (store *Store[K, V]) Delete(key K) bool {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	if _, ok := store.items[key]; !ok {
		return false
	}

	delete(store.items, key)
	return true
}
//
// func converterToOrder[V any](value any) enum.Order{
// 	order := enum.Order{
// 		ID:       value.UUID,
// 		TraderID:  value.UUID,
// 		Side:      value.Side,
// 		Type:      value.OrderType,
// 		Symbo:     value.Symbo,
// 		Price:     value.Price,
// 		Quantity:  value.Quantity,
// 		Filled:    value.Filled,
// 		Status:    value.Status,
// 		CreatedAt: value.CreatedAt,
// 		UpdatedAt: value.UpdatedAt,
// 	}
// 	return order
// }

func convertToOrder(value any) enum.Order{
	return value.(enum.Order)
}
// атомарный read-modify-write под  Lock()
func (store *Store[K, V]) Update(key K, modif func(V) V) (V, bool) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	value, ok := store.items[key]
	if !ok {
		var zero V
		return zero, false
	}
	
	Updatedvalue := modif(value)
	// после update нужно сохранить все в store
	store.items[key] = Updatedvalue 
	updatedOrder := convertToOrder(Updatedvalue)
	enum.LogOrderState(&updatedOrder)
	return Updatedvalue, true
}

type IndexedStore[K comparable, V any] struct {
	// Тербует уточнения
	mutex sync.RWMutex
	order []K
	byKey map[K]int // позволяет за О(1) узнать проядок в срезе
	data  map[K]V   // хранилище значение по ключу
}

func NewIndexedStore[K comparable, V any](cap int) *IndexedStore[K, V] {
	return &IndexedStore[K, V]{
		order: make([]K, 0, cap),
		byKey: make(map[K]int, cap),
		data:  make(map[K]V, cap),
	}
}

// Put добавляет пару или обновляет заначение по ключу.
// При обновлении порядок вставки не меняется.
func (storage *IndexedStore[K, V]) Put(key K, newValue V) {
	storage.mutex.Lock()         // блокируем на время операции
	defer storage.mutex.Unlock() // гарантированно разблокируем

	// Ключ уже есть - обновляем значение, order/byKey не трогаем
	if _, ok := storage.data[key]; ok {
		storage.data[key] = newValue
		// fmt.Println("Значение изменено с".value . "")
		return
	}
	// новый ключ
	storage.byKey[key] = len(storage.order)
	// Добавляем ключ в конец - фиксируем порядок вставки
	storage.order = append(storage.order, key)
	// Сохраняем значение
	storage.data[key] = newValue
}

// возвращает значение в пордке вставки под RLock()
func (indexStore *IndexedStore[K, V]) Ordered() []V {
	indexStore.mutex.RLock()
	defer indexStore.mutex.RUnlock()

	out := make([]V, len(indexStore.order))
	for index, value := range indexStore.order {
		out[index] = indexStore.data[value]
	}
	return out
}

// Описание универсального множества
type Storage[T comparable] struct {
	storage map[T]struct{}
} // хранение универсальных множеств

func NewStorage[T comparable](cap int) *Storage[T] {
	return &Storage[T]{
		storage: make(map[T]struct{}, cap),
	}
}

func (storage *Storage[T]) Set(value T) {
	storage.storage[value] = struct{}{}
}

func (storage *Storage[T]) Remove(value T) {
	delete(storage.storage, value)
}

func (storage *Storage[T]) Has(value T) bool {
	_, ok := storage.storage[value]
	return ok
}

func (storage *Storage[T]) Lenght() int {
	return len(storage.storage)
}

func (storage *Storage[T]) Items() []T {
	outSlice := make([]T, 0, len(storage.storage))
	for value := range storage.storage {
		outSlice = append(outSlice, value)
	}
	return outSlice
}

// Новый строчка кода: функция сделки
func(storage *IndexedStore[K, V]) UpdatePair(key1, key2 K, modify func(V, V) (V, V, error),) error {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	value1, ok1 := storage.data[key1]
	value2, ok2 := storage.data[key2]

	if !ok1 || !ok2{
		return fmt.Errorf("один из ордеров не найден")
	}

	newValue1, newValue2, err := modify(value1, value2)
	if err != nil {
		return err
	}

	storage.data[key1] = newValue1
	storage.data[key2] = newValue2

	return nil
}

func (storage *IndexedStore[K, V]) Delete(key K) bool {
	storage.mutex.Lock()         // блокируем на время операции
	defer storage.mutex.Unlock() // гарантированно разблокируем

	// Ключ уже есть - обновляем значение, order/byKey не трогаем
	indexInSlice, ok := storage.byKey[key]
	if !ok {
		return false
	}

	lastIndex := len(storage.order) - 1

	if indexInSlice != lastIndex {
		lastValue := storage.order[lastIndex]
		storage.order[indexInSlice] = lastValue
		storage.byKey[lastValue] = indexInSlice
	}

	// отбрасываем последний элемент слайса
	storage.order = storage.order[:lastIndex]
	// удаляем ключ из обратного индекса и основного хранилища
	delete(storage.byKey, key)
	delete(storage.data, key)

	return true
}
