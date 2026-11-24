package analytics

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Default store timezone for merchant analytics (PRD §31, §12).
const DefaultTZ = "Asia/Jakarta"

type Overview struct {
	Timezone               string          `json:"timezone"`
	From                   string          `json:"from"` // YYYY-MM-DD in store TZ
	To                     string          `json:"to"`
	TotalOrders            int             `json:"total_orders"`
	PaidOrders             int             `json:"paid_orders"`
	VerifiedPaymentValueIDR int64          `json:"verified_payment_value_idr"`
	UnpaidOrderValueIDR    int64           `json:"unpaid_order_value_idr"`
	TotalOrderValueIDR     int64           `json:"total_order_value_idr"`
	UniqueCustomers        int             `json:"unique_customers"`
	ConversionRate         float64         `json:"conversion_rate"` // paid / total orders in range
	Daily                  []DailyPoint    `json:"daily"`
	TopProducts            []TopProduct    `json:"top_products"`
}

type DailyPoint struct {
	Date              string `json:"date"`
	Orders            int    `json:"orders"`
	PaidOrders        int    `json:"paid_orders"`
	PaidValueIDR      int64  `json:"paid_value_idr"`
	OrderValueIDR     int64  `json:"order_value_idr"`
}

type TopProduct struct {
	ProductID   uuid.UUID `json:"product_id"`
	ProductName string    `json:"product_name"`
	Quantity    int       `json:"quantity"`
	RevenueIDR  int64     `json:"revenue_idr"`
}

type Service struct {
	DB *pgxpool.Pool
}

func loadLocation(tz string) *time.Location {
	if tz == "" {
		tz = DefaultTZ
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc, _ = time.LoadLocation(DefaultTZ)
	}
	return loc
}

// rangeBounds returns UTC [start, endExclusive) covering inclusive local dates from/to.
func rangeBounds(from, to string, loc *time.Location) (time.Time, time.Time, string, string, error) {
	nowLocal := time.Now().In(loc)
	if to == "" {
		to = nowLocal.Format("2006-01-02")
	}
	if from == "" {
		from = nowLocal.AddDate(0, 0, -29).Format("2006-01-02")
	}
	startLocal, err := time.ParseInLocation("2006-01-02", from, loc)
	if err != nil {
		return time.Time{}, time.Time{}, "", "", err
	}
	endLocal, err := time.ParseInLocation("2006-01-02", to, loc)
	if err != nil {
		return time.Time{}, time.Time{}, "", "", err
	}
	endExclusive := endLocal.AddDate(0, 0, 1)
	return startLocal.UTC(), endExclusive.UTC(), from, to, nil
}

func (s *Service) Overview(ctx context.Context, storeID uuid.UUID, from, to, tz string) (*Overview, error) {
	if tz == "" {
		_ = s.DB.QueryRow(ctx, `SELECT COALESCE(NULLIF(timezone,''), $2) FROM stores WHERE id=$1`, storeID, DefaultTZ).Scan(&tz)
		if tz == "" {
			tz = DefaultTZ
		}
	}
	loc := loadLocation(tz)
	startUTC, endUTC, fromS, toS, err := rangeBounds(from, to, loc)
	if err != nil {
		return nil, err
	}

	out := &Overview{Timezone: tz, From: fromS, To: toS}

	err = s.DB.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE order_status NOT IN ('cancelled')),
			COUNT(*) FILTER (WHERE payment_status='paid' AND order_status NOT IN ('cancelled')),
			COALESCE(SUM(total_idr) FILTER (WHERE payment_status='paid' AND order_status NOT IN ('cancelled')),0),
			COALESCE(SUM(total_idr) FILTER (
				WHERE payment_status IN ('unpaid','awaiting_verification')
				  AND order_status NOT IN ('cancelled')
			),0),
			COALESCE(SUM(total_idr) FILTER (WHERE order_status NOT IN ('cancelled')),0),
			COUNT(DISTINCT customer_id) FILTER (WHERE order_status NOT IN ('cancelled') AND customer_id IS NOT NULL)
		FROM orders
		WHERE store_id=$1 AND created_at >= $2 AND created_at < $3
	`, storeID, startUTC, endUTC).Scan(
		&out.TotalOrders, &out.PaidOrders, &out.VerifiedPaymentValueIDR,
		&out.UnpaidOrderValueIDR, &out.TotalOrderValueIDR, &out.UniqueCustomers,
	)
	if err != nil {
		return nil, err
	}
	if out.TotalOrders > 0 {
		out.ConversionRate = float64(out.PaidOrders) / float64(out.TotalOrders)
	}

	rows, err := s.DB.Query(ctx, `
		SELECT to_char(timezone($4, created_at), 'YYYY-MM-DD') AS d,
		       COUNT(*) FILTER (WHERE order_status NOT IN ('cancelled')),
		       COUNT(*) FILTER (WHERE payment_status='paid' AND order_status NOT IN ('cancelled')),
		       COALESCE(SUM(total_idr) FILTER (WHERE payment_status='paid' AND order_status NOT IN ('cancelled')),0),
		       COALESCE(SUM(total_idr) FILTER (WHERE order_status NOT IN ('cancelled')),0)
		FROM orders
		WHERE store_id=$1 AND created_at >= $2 AND created_at < $3
		GROUP BY 1
		ORDER BY 1
	`, storeID, startUTC, endUTC, tz)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p DailyPoint
		if err := rows.Scan(&p.Date, &p.Orders, &p.PaidOrders, &p.PaidValueIDR, &p.OrderValueIDR); err != nil {
			return nil, err
		}
		out.Daily = append(out.Daily, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	prows, err := s.DB.Query(ctx, `
		SELECT oi.product_id, MAX(oi.product_name),
		       COALESCE(SUM(oi.quantity),0),
		       COALESCE(SUM(oi.line_total_idr),0)
		FROM order_items oi
		JOIN orders o ON o.id=oi.order_id AND o.store_id=oi.store_id
		WHERE oi.store_id=$1
		  AND o.created_at >= $2 AND o.created_at < $3
		  AND o.order_status NOT IN ('cancelled')
		  AND o.payment_status='paid'
		GROUP BY oi.product_id
		ORDER BY SUM(oi.line_total_idr) DESC
		LIMIT 10
	`, storeID, startUTC, endUTC)
	if err != nil {
		return nil, err
	}
	defer prows.Close()
	for prows.Next() {
		var tp TopProduct
		if err := prows.Scan(&tp.ProductID, &tp.ProductName, &tp.Quantity, &tp.RevenueIDR); err != nil {
			return nil, err
		}
		out.TopProducts = append(out.TopProducts, tp)
	}
	if out.Daily == nil {
		out.Daily = []DailyPoint{}
	}
	if out.TopProducts == nil {
		out.TopProducts = []TopProduct{}
	}
	return out, prows.Err()
}
