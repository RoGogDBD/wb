package repository

import (
	"context"
	"fmt"
	"log"

	"github.com/RoGogDBD/wb/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresStorage(pool *pgxpool.Pool) *PostgresStorage {
	return &PostgresStorage{pool: pool}
}

func (r *PostgresStorage) InsertOrder(ctx context.Context, o *models.Order) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	orderUUID, err := uuid.Parse(o.OrderUID)
	if err != nil {
		return fmt.Errorf("invalid UUID: %w", err)
	}

	// orders
	_, err = tx.Exec(ctx, `
        INSERT INTO orders (order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
        ON CONFLICT (order_uid) DO UPDATE
        SET track_number = EXCLUDED.track_number,
            entry = EXCLUDED.entry,
            locale = EXCLUDED.locale,
            internal_signature = EXCLUDED.internal_signature,
            customer_id = EXCLUDED.customer_id,
            delivery_service = EXCLUDED.delivery_service,
            shardkey = EXCLUDED.shardkey,
            sm_id = EXCLUDED.sm_id,
            date_created = EXCLUDED.date_created,
            oof_shard = EXCLUDED.oof_shard;
    `, orderUUID, o.TrackNumber, o.Entry, o.Locale, o.InternalSignature, o.CustomerID, o.DeliveryService, o.ShardKey, o.SmID, o.DateCreated, o.OofShard)
	if err != nil {
		return fmt.Errorf("insert orders: %w", err)
	}

	// delivery
	_, err = tx.Exec(ctx, `
        INSERT INTO deliveries (order_uid, name, phone, zip, city, address, region, email)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
        ON CONFLICT (order_uid) DO UPDATE
        SET name=EXCLUDED.name, phone=EXCLUDED.phone, zip=EXCLUDED.zip,
            city=EXCLUDED.city, address=EXCLUDED.address,
            region=EXCLUDED.region, email=EXCLUDED.email;
    `, orderUUID, o.Delivery.Name, o.Delivery.Phone, o.Delivery.Zip, o.Delivery.City, o.Delivery.Address, o.Delivery.Region, o.Delivery.Email)
	if err != nil {
		return fmt.Errorf("insert delivery: %w", err)
	}

	// payment
	_, err = tx.Exec(ctx, `
        INSERT INTO payments (order_uid, transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
        ON CONFLICT (order_uid) DO UPDATE
        SET transaction=EXCLUDED.transaction, request_id=EXCLUDED.request_id,
            currency=EXCLUDED.currency, provider=EXCLUDED.provider,
            amount=EXCLUDED.amount, payment_dt=EXCLUDED.payment_dt,
            bank=EXCLUDED.bank, delivery_cost=EXCLUDED.delivery_cost,
            goods_total=EXCLUDED.goods_total, custom_fee=EXCLUDED.custom_fee;
    `, orderUUID, o.Payment.Transaction, o.Payment.RequestID, o.Payment.Currency, o.Payment.Provider,
		o.Payment.Amount, o.Payment.PaymentDt, o.Payment.Bank, o.Payment.DeliveryCost,
		o.Payment.GoodsTotal, o.Payment.CustomFee)
	if err != nil {
		return fmt.Errorf("insert payment: %w", err)
	}

	// items
	_, err = tx.Exec(ctx, `DELETE FROM items WHERE order_uid=$1`, orderUUID)
	if err != nil {
		return fmt.Errorf("delete items: %w", err)
	}

	for _, it := range o.Items {
		_, err = tx.Exec(ctx, `
            INSERT INTO items (order_uid, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status)
            VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
        `, orderUUID, it.ChrtID, it.TrackNumber, it.Price, it.Rid, it.Name, it.Sale, it.Size, it.TotalPrice, it.NmID, it.Brand, it.Status)
		if err != nil {
			return fmt.Errorf("insert item: %w", err)
		}
	}

	// commit
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (r *PostgresStorage) GetOrderByID(ctx context.Context, orderUID string) (*models.Order, error) {
	o := &models.Order{}
	row := r.pool.QueryRow(ctx, `
		SELECT order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
		FROM orders WHERE order_uid=$1
	`, orderUID)
	if err := row.Scan(&o.OrderUID, &o.TrackNumber, &o.Entry, &o.Locale, &o.InternalSignature,
		&o.CustomerID, &o.DeliveryService, &o.ShardKey, &o.SmID, &o.DateCreated, &o.OofShard); err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}

	row = r.pool.QueryRow(ctx, `
		SELECT name, phone, zip, city, address, region, email
		FROM deliveries WHERE order_uid=$1
	`, orderUID)
	if err := row.Scan(&o.Delivery.Name, &o.Delivery.Phone, &o.Delivery.Zip,
		&o.Delivery.City, &o.Delivery.Address, &o.Delivery.Region, &o.Delivery.Email); err != nil {
		return nil, fmt.Errorf("get delivery: %w", err)
	}

	row = r.pool.QueryRow(ctx, `
		SELECT transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee
		FROM payments WHERE order_uid=$1
	`, orderUID)
	if err := row.Scan(&o.Payment.Transaction, &o.Payment.RequestID, &o.Payment.Currency,
		&o.Payment.Provider, &o.Payment.Amount, &o.Payment.PaymentDt, &o.Payment.Bank,
		&o.Payment.DeliveryCost, &o.Payment.GoodsTotal, &o.Payment.CustomFee); err != nil {
		return nil, fmt.Errorf("get payment: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status
		FROM items WHERE order_uid=$1
	`, orderUID)
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

func (r *PostgresStorage) GetAllOrders(ctx context.Context) ([]models.Order, error) {
	var orders []models.Order

	rows, err := r.pool.Query(ctx, `SELECT order_uid FROM orders`)
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
