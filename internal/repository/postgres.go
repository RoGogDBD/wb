package repository

import (
	"context"
	"errors"
	"fmt"
	"log"

	sq "github.com/Masterminds/squirrel"
	"github.com/RoGogDBD/wb/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStorage хранит заказы в PostgreSQL.
type PostgresStorage struct {
	pool *pgxpool.Pool
}

// NewPostgresStorage создает PostgresStorage.
func NewPostgresStorage(pool *pgxpool.Pool) *PostgresStorage {
	return &PostgresStorage{pool: pool}
}

// InsertOrder выполняет вставку или обновление заказа и связанных данных.
func (r *PostgresStorage) InsertOrder(ctx context.Context, o *models.Order) error {
	builder := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			log.Printf("rollback failed: %v", err)
		}
	}()

	orderUUID, err := uuid.Parse(o.OrderUID)
	if err != nil {
		return fmt.Errorf("invalid UUID: %w", err)
	}

	// заказы
	orderSQL, orderArgs, err := builder.Insert("orders").
		Columns(
			"order_uid",
			"track_number",
			"entry",
			"locale",
			"internal_signature",
			"customer_id",
			"delivery_service",
			"shardkey",
			"sm_id",
			"date_created",
			"oof_shard",
		).
		Values(
			orderUUID,
			o.TrackNumber,
			o.Entry,
			o.Locale,
			o.InternalSignature,
			o.CustomerID,
			o.DeliveryService,
			o.ShardKey,
			o.SmID,
			o.DateCreated,
			o.OofShard,
		).
		Suffix(`ON CONFLICT (order_uid) DO UPDATE
        SET track_number = EXCLUDED.track_number,
            entry = EXCLUDED.entry,
            locale = EXCLUDED.locale,
            internal_signature = EXCLUDED.internal_signature,
            customer_id = EXCLUDED.customer_id,
            delivery_service = EXCLUDED.delivery_service,
            shardkey = EXCLUDED.shardkey,
            sm_id = EXCLUDED.sm_id,
            date_created = EXCLUDED.date_created,
            oof_shard = EXCLUDED.oof_shard`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build orders insert: %w", err)
	}
	_, err = tx.Exec(ctx, orderSQL, orderArgs...)
	if err != nil {
		return fmt.Errorf("insert orders: %w", err)
	}

	// доставка
	deliverySQL, deliveryArgs, err := builder.Insert("deliveries").
		Columns("order_uid", "name", "phone", "zip", "city", "address", "region", "email").
		Values(
			orderUUID,
			o.Delivery.Name,
			o.Delivery.Phone,
			o.Delivery.Zip,
			o.Delivery.City,
			o.Delivery.Address,
			o.Delivery.Region,
			o.Delivery.Email,
		).
		Suffix(`ON CONFLICT (order_uid) DO UPDATE
        SET name=EXCLUDED.name, phone=EXCLUDED.phone, zip=EXCLUDED.zip,
            city=EXCLUDED.city, address=EXCLUDED.address,
            region=EXCLUDED.region, email=EXCLUDED.email`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delivery insert: %w", err)
	}
	_, err = tx.Exec(ctx, deliverySQL, deliveryArgs...)
	if err != nil {
		return fmt.Errorf("insert delivery: %w", err)
	}

	// оплата
	paymentSQL, paymentArgs, err := builder.Insert("payments").
		Columns(
			"order_uid",
			"transaction",
			"request_id",
			"currency",
			"provider",
			"amount",
			"payment_dt",
			"bank",
			"delivery_cost",
			"goods_total",
			"custom_fee",
		).
		Values(
			orderUUID,
			o.Payment.Transaction,
			o.Payment.RequestID,
			o.Payment.Currency,
			o.Payment.Provider,
			o.Payment.Amount,
			o.Payment.PaymentDt,
			o.Payment.Bank,
			o.Payment.DeliveryCost,
			o.Payment.GoodsTotal,
			o.Payment.CustomFee,
		).
		Suffix(`ON CONFLICT (order_uid) DO UPDATE
        SET transaction=EXCLUDED.transaction, request_id=EXCLUDED.request_id,
            currency=EXCLUDED.currency, provider=EXCLUDED.provider,
            amount=EXCLUDED.amount, payment_dt=EXCLUDED.payment_dt,
            bank=EXCLUDED.bank, delivery_cost=EXCLUDED.delivery_cost,
            goods_total=EXCLUDED.goods_total, custom_fee=EXCLUDED.custom_fee`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build payment insert: %w", err)
	}
	_, err = tx.Exec(ctx, paymentSQL, paymentArgs...)
	if err != nil {
		return fmt.Errorf("insert payment: %w", err)
	}

	// товары
	deleteSQL, deleteArgs, err := builder.Delete("items").
		Where(sq.Eq{"order_uid": orderUUID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete items: %w", err)
	}
	_, err = tx.Exec(ctx, deleteSQL, deleteArgs...)
	if err != nil {
		return fmt.Errorf("delete items: %w", err)
	}

	for _, it := range o.Items {
		itemSQL, itemArgs, err := builder.Insert("items").
			Columns(
				"order_uid",
				"chrt_id",
				"track_number",
				"price",
				"rid",
				"name",
				"sale",
				"size",
				"total_price",
				"nm_id",
				"brand",
				"status",
			).
			Values(
				orderUUID,
				it.ChrtID,
				it.TrackNumber,
				it.Price,
				it.Rid,
				it.Name,
				it.Sale,
				it.Size,
				it.TotalPrice,
				it.NmID,
				it.Brand,
				it.Status,
			).
			ToSql()
		if err != nil {
			return fmt.Errorf("build insert item: %w", err)
		}
		_, err = tx.Exec(ctx, itemSQL, itemArgs...)
		if err != nil {
			return fmt.Errorf("insert item: %w", err)
		}
	}

	// фиксация транзакции
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// GetOrderByID загружает заказ по ID.
func (r *PostgresStorage) GetOrderByID(ctx context.Context, orderUID string) (*models.Order, error) {
	builder := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	o := &models.Order{}
	orderSQL, orderArgs, err := builder.Select(
		"order_uid",
		"track_number",
		"entry",
		"locale",
		"internal_signature",
		"customer_id",
		"delivery_service",
		"shardkey",
		"sm_id",
		"date_created",
		"oof_shard",
	).
		From("orders").
		Where(sq.Eq{"order_uid": orderUID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get order: %w", err)
	}
	row := r.pool.QueryRow(ctx, orderSQL, orderArgs...)
	if err := row.Scan(&o.OrderUID, &o.TrackNumber, &o.Entry, &o.Locale, &o.InternalSignature,
		&o.CustomerID, &o.DeliveryService, &o.ShardKey, &o.SmID, &o.DateCreated, &o.OofShard); err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}

	deliverySQL, deliveryArgs, err := builder.Select(
		"name",
		"phone",
		"zip",
		"city",
		"address",
		"region",
		"email",
	).
		From("deliveries").
		Where(sq.Eq{"order_uid": orderUID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get delivery: %w", err)
	}
	row = r.pool.QueryRow(ctx, deliverySQL, deliveryArgs...)
	if err := row.Scan(&o.Delivery.Name, &o.Delivery.Phone, &o.Delivery.Zip,
		&o.Delivery.City, &o.Delivery.Address, &o.Delivery.Region, &o.Delivery.Email); err != nil {
		return nil, fmt.Errorf("get delivery: %w", err)
	}

	paymentSQL, paymentArgs, err := builder.Select(
		"transaction",
		"request_id",
		"currency",
		"provider",
		"amount",
		"payment_dt",
		"bank",
		"delivery_cost",
		"goods_total",
		"custom_fee",
	).
		From("payments").
		Where(sq.Eq{"order_uid": orderUID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get payment: %w", err)
	}
	row = r.pool.QueryRow(ctx, paymentSQL, paymentArgs...)
	if err := row.Scan(&o.Payment.Transaction, &o.Payment.RequestID, &o.Payment.Currency,
		&o.Payment.Provider, &o.Payment.Amount, &o.Payment.PaymentDt, &o.Payment.Bank,
		&o.Payment.DeliveryCost, &o.Payment.GoodsTotal, &o.Payment.CustomFee); err != nil {
		return nil, fmt.Errorf("get payment: %w", err)
	}

	itemsSQL, itemsArgs, err := builder.Select(
		"chrt_id",
		"track_number",
		"price",
		"rid",
		"name",
		"sale",
		"size",
		"total_price",
		"nm_id",
		"brand",
		"status",
	).
		From("items").
		Where(sq.Eq{"order_uid": orderUID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get items: %w", err)
	}
	rows, err := r.pool.Query(ctx, itemsSQL, itemsArgs...)
	if err != nil {
		return nil, fmt.Errorf("get items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var it models.Item
		if err := rows.Scan(&it.ChrtID, &it.TrackNumber, &it.Price, &it.Rid, &it.Name,
			&it.Sale, &it.Size, &it.TotalPrice, &it.NmID, &it.Brand, &it.Status); err != nil {
			return nil, fmt.Errorf("scan item: %w", err)
		}
		o.Items = append(o.Items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan items rows: %w", err)
	}

	return o, nil
}

// GetAllOrders загружает все заказы.
func (r *PostgresStorage) GetAllOrders(ctx context.Context) ([]models.Order, error) {
	builder := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	var orders []models.Order

	allSQL, allArgs, err := builder.Select("order_uid").From("orders").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query all orders: %w", err)
	}
	rows, err := r.pool.Query(ctx, allSQL, allArgs...)
	if err != nil {
		return nil, fmt.Errorf("query all orders: %w", err)
	}
	defer rows.Close()

	var orderUIDs []string
	for rows.Next() {
		var orderUID string
		if err := rows.Scan(&orderUID); err != nil {
			return nil, fmt.Errorf("scan order_uid: %w", err)
		}
		orderUIDs = append(orderUIDs, orderUID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan order_uid rows: %w", err)
	}

	for _, uid := range orderUIDs {
		order, err := r.GetOrderByID(ctx, uid)
		if err != nil {
			log.Printf("Warning: failed to load order %s: %v", uid, err)
			continue
		}
		orders = append(orders, *order)
	}

	return orders, nil
}
