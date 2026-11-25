package storage

import (
	"fmt"
	"hash/fnv"
	"sync"

	"github.com/google/uuid"

	"enump/enum"
)

// Store — универсальное потокобезопасное хранилище.
type Store[K comparable, V any] struct {
	mutex sync.RWMutex
	items map[K]V
}

func NewStore[K comparable, V any](capacity int) *Store[K, V] {
	return &Store[K, V]{
		items: make(map[K]V, capacity),
	}
}

// Put добавляет или обновляет значение.
func (store *Store[K, V]) Put(key K, value V) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	store.items[key] = value
}

// Get получает значение по ключу.
func (store *Store[K, V]) Get(key K) (V, bool) {
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	value, ok := store.items[key]
	return value, ok
}

// Delete удаляет значение по ключу.
func (store *Store[K, V]) Delete(key K) bool {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	if _, ok := store.items[key]; !ok {
		return false
	}

	delete(store.items, key)
	return true
}

// Update атомарно изменяет значение по ключу.
func (store *Store[K, V]) Update(
	key K,
	modify func(V) V,
) (V, bool) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	value, ok := store.items[key]
	if !ok {
		var zero V
		return zero, false
	}

	updatedValue := modify(value)
	store.items[key] = updatedValue

	return updatedValue, true
}

// IndexedStore — хранилище с сохранением порядка вставки.
type IndexedStore[K comparable, V any] struct {
	mutex sync.RWMutex

	order []K
	byKey map[K]int
	data  map[K]V
}

func NewIndexedStore[K comparable, V any](capacity int) *IndexedStore[K, V] {
	return &IndexedStore[K, V]{
		order: make([]K, 0, capacity),
		byKey: make(map[K]int, capacity),
		data:  make(map[K]V, capacity),
	}
}

// Put добавляет новую запись или обновляет существующую.
// При обновлении порядок вставки не меняется.
func (storage *IndexedStore[K, V]) Put(key K, newValue V) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	if _, ok := storage.data[key]; ok {
		storage.data[key] = newValue
		return
	}

	storage.byKey[key] = len(storage.order)
	storage.order = append(storage.order, key)
	storage.data[key] = newValue
}

// Get получает значение по ключу.
func (storage *IndexedStore[K, V]) Get(key K) (V, bool) {
	storage.mutex.RLock()
	defer storage.mutex.RUnlock()

	value, ok := storage.data[key]
	return value, ok
}

// Ordered возвращает значения в порядке вставки.
// После удаления порядок может измениться, поскольку Delete использует swap-with-last.
func (storage *IndexedStore[K, V]) Ordered() []V {
	storage.mutex.RLock()
	defer storage.mutex.RUnlock()

	out := make([]V, 0, len(storage.order))

	for _, key := range storage.order {
		out = append(out, storage.data[key])
	}

	return out
}

// UpdatePair атомарно обновляет две записи.
func (storage *IndexedStore[K, V]) UpdatePair(
	key1, key2 K,
	modify func(V, V) (V, V, error),
) error {
	if key1 == key2 {
		return fmt.Errorf("нельзя обновить одну и ту же запись дважды")
	}

	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	value1, ok1 := storage.data[key1]
	value2, ok2 := storage.data[key2]

	if !ok1 || !ok2 {
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

// Delete удаляет запись.
// Используется swap-with-last для удаления за O(1).
func (storage *IndexedStore[K, V]) Delete(key K) bool {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	indexInSlice, ok := storage.byKey[key]
	if !ok {
		return false
	}

	lastIndex := len(storage.order) - 1

	if indexInSlice != lastIndex {
		lastKey := storage.order[lastIndex]

		storage.order[indexInSlice] = lastKey
		storage.byKey[lastKey] = indexInSlice
	}

	storage.order = storage.order[:lastIndex]

	delete(storage.byKey, key)
	delete(storage.data, key)

	return true
}

// Storage — универсальное множество.
type Storage[T comparable] struct {
	mutex   sync.RWMutex
	storage map[T]struct{}
}

func NewStorage[T comparable](capacity int) *Storage[T] {
	return &Storage[T]{
		storage: make(map[T]struct{}, capacity),
	}
}

// Set добавляет элемент в множество.
func (storage *Storage[T]) Set(value T) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	storage.storage[value] = struct{}{}
}

// Remove удаляет элемент из множества.
func (storage *Storage[T]) Remove(value T) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	delete(storage.storage, value)
}

// Has проверяет наличие элемента.
func (storage *Storage[T]) Has(value T) bool {
	storage.mutex.RLock()
	defer storage.mutex.RUnlock()

	_, ok := storage.storage[value]
	return ok
}

// Length возвращает количество элементов.
func (storage *Storage[T]) Length() int {
	storage.mutex.RLock()
	defer storage.mutex.RUnlock()

	return len(storage.storage)
}

// Lenght оставлен для совместимости со старым названием.
func (storage *Storage[T]) Lenght() int {
	return storage.Length()
}

// Items возвращает все элементы множества.
func (storage *Storage[T]) Items() []T {
	storage.mutex.RLock()
	defer storage.mutex.RUnlock()

	out := make([]T, 0, len(storage.storage))

	for value := range storage.storage {
		out = append(out, value)
	}

	return out
}

// ShardedOrderBook — шардированное хранилище ордеров.
// Ордеры одной торговой пары попадают в один шард.
type ShardedOrderBook struct {
	shards []*IndexedStore[uuid.UUID, enum.Order]
}

func NewShardedOrderBook(
	shardCount int,
	capacity int,
) *ShardedOrderBook {
	if shardCount <= 0 {
		panic("shardCount должен быть больше нуля")
	}

	shards := make([]*IndexedStore[uuid.UUID, enum.Order], shardCount)

	for i := range shards {
		shards[i] = NewIndexedStore[uuid.UUID, enum.Order](capacity)
	}

	return &ShardedOrderBook{
		shards: shards,
	}
}

// shardIndex определяет номер шарда по торговой паре.
func (book *ShardedOrderBook) shardIndex(symbo string) int {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(symbo))

	return int(hash.Sum32() % uint32(len(book.shards)))
}

// getShard возвращает нужный шард.
func (book *ShardedOrderBook) getShard(
	symbo string,
) *IndexedStore[uuid.UUID, enum.Order] {
	return book.shards[book.shardIndex(symbo)]
}

// Put добавляет ордер в соответствующий шард.
func (book *ShardedOrderBook) Put(order enum.Order) error {
	if order.Symbo == "" {
		return fmt.Errorf("symbo не может быть пустым")
	}

	book.getShard(order.Symbo).Put(order.ID, order)

	return nil
}

// Get получает ордер по торговой паре и ID.
func (book *ShardedOrderBook) Get(
	symbo string,
	id uuid.UUID,
) (enum.Order, bool) {
	return book.getShard(symbo).Get(id)
}

// Delete удаляет ордер по торговой паре и ID.
func (book *ShardedOrderBook) Delete(
	symbo string,
	id uuid.UUID,
) bool {
	return book.getShard(symbo).Delete(id)
}

// UpdatePair обновляет два ордера одной торговой пары.
func (book *ShardedOrderBook) UpdatePair(
	symbo string,
	id1 uuid.UUID,
	id2 uuid.UUID,
	modify func(enum.Order, enum.Order) (enum.Order, enum.Order, error),
) error {
	if symbo == "" {
		return fmt.Errorf("symbo не может быть пустым")
	}

	return book.getShard(symbo).UpdatePair(id1, id2, func(order1, order2 enum.Order) (enum.Order, enum.Order, error) {
		if order1.Symbo != symbo || order2.Symbo != symbo {
			return order1, order2, fmt.Errorf(
				"ордера принадлежат другой торговой паре",
			)
		}

		return modify(order1, order2)
	},
	)
}
