package commerce

import (
	"context"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type PurchasingFilter struct {
	BatchID    *uuid.UUID
	CampaignID *uuid.UUID
	VendorID   *uuid.UUID
	BrandID    *uuid.UUID
	GroupBy    string // variant (default) | vendor | brand | campaign
}

type PurchasingEligibility struct {
	BatchID            uuid.UUID  `json:"batch_id"`
	BatchName          string     `json:"batch_name"`
	BatchSlug          string     `json:"batch_slug"`
	OperationalStatus  string     `json:"operational_status"`
	CampaignID         *uuid.UUID `json:"campaign_id,omitempty"`
	CampaignName       *string    `json:"campaign_name,omitempty"`
	PaidOrderCount     int        `json:"paid_order_count"`
	UnpaidOrderCount   int        `json:"unpaid_order_count"`
	RequiredLineQty    int        `json:"required_line_qty"`
	OutstandingLineQty int        `json:"outstanding_line_qty"`
	PurchasingEligible bool       `json:"purchasing_eligible"`
}

// ListPurchasingEligibility returns batches/campaigns ready for purchasing work.
func (s *Service) ListPurchasingEligibility(ctx context.Context, storeID uuid.UUID) ([]PurchasingEligibility, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT b.id, b.name, b.slug, b.operational_status, b.campaign_id, c.name,
		       COALESCE((
		         SELECT COUNT(*) FROM orders o
		         WHERE o.store_id=b.store_id AND o.preorder_batch_id=b.id
		           AND o.order_status IN ('confirmed','completed')
		           AND o.payment_status='paid'
		       ),0) AS paid_orders,
		       COALESCE((
		         SELECT COUNT(*) FROM orders o
		         WHERE o.store_id=b.store_id AND o.preorder_batch_id=b.id
		           AND o.order_status NOT IN ('cancelled')
		           AND o.payment_status IN ('unpaid','awaiting_verification')
		       ),0) AS unpaid_orders,
		       COALESCE((
		         SELECT SUM(oi.quantity) FROM order_items oi
		         JOIN orders o ON o.id=oi.order_id AND o.store_id=oi.store_id
		         WHERE oi.store_id=b.store_id AND o.preorder_batch_id=b.id
		           AND o.order_status IN ('confirmed','completed')
		           AND o.payment_status='paid'
		       ),0) AS required_qty
		FROM preorder_batches b
		LEFT JOIN campaigns c ON c.id=b.campaign_id AND c.store_id=b.store_id
		WHERE b.store_id=$1
		  AND b.operational_status IN (
		    'closed','purchasing','awaiting_arrival','arrived','fulfilling','completed'
		  )
		ORDER BY b.closes_at DESC, b.name
	`, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PurchasingEligibility
	for rows.Next() {
		var e PurchasingEligibility
		if err := rows.Scan(
			&e.BatchID, &e.BatchName, &e.BatchSlug, &e.OperationalStatus,
			&e.CampaignID, &e.CampaignName,
			&e.PaidOrderCount, &e.UnpaidOrderCount, &e.RequiredLineQty,
		); err != nil {
			return nil, err
		}
		var purchased int
		_ = s.DB.QueryRow(ctx, `
			SELECT COALESCE(SUM(purchased_qty),0) FROM preorder_purchasing_records
			WHERE store_id=$1 AND batch_id=$2
		`, storeID, e.BatchID).Scan(&purchased)
		outQty := e.RequiredLineQty - purchased
		if outQty < 0 {
			outQty = 0
		}
		e.OutstandingLineQty = outQty
		e.PurchasingEligible = e.RequiredLineQty > 0 || e.UnpaidOrderCount > 0 ||
			e.OperationalStatus == "purchasing" || e.OperationalStatus == "awaiting_arrival"
		out = append(out, e)
	}
	return out, rows.Err()
}

// AggregatePurchasing consolidates demand by exact variant (default), optionally
// filtered by campaign / batch / vendor / brand (PUR-01, PUR-04, PUR-05, VEN-03).
func (s *Service) AggregatePurchasing(ctx context.Context, storeID uuid.UUID, f PurchasingFilter) ([]PurchasingLine, error) {
	groupBy := strings.ToLower(strings.TrimSpace(f.GroupBy))
	if groupBy == "" {
		groupBy = "variant"
	}

	args := []any{storeID}
	where := []string{"oi.store_id=$1", "o.order_status NOT IN ('cancelled')", "o.preorder_batch_id IS NOT NULL"}
	argN := 2
	if f.BatchID != nil {
		where = append(where, "o.preorder_batch_id=$"+strconv.Itoa(argN))
		args = append(args, *f.BatchID)
		argN++
	}
	if f.CampaignID != nil {
		where = append(where, "(o.campaign_id=$"+strconv.Itoa(argN)+" OR b.campaign_id=$"+strconv.Itoa(argN)+")")
		args = append(args, *f.CampaignID)
		argN++
	}
	if f.BrandID != nil {
		where = append(where, "p.brand_id=$"+strconv.Itoa(argN))
		args = append(args, *f.BrandID)
		argN++
	}

	whereSQL := strings.Join(where, " AND ")

	// Always aggregate demand at exact variant grain; enrich with preferred vendor / brand.
	q := `
		SELECT oi.product_id, oi.variant_id,
		       MAX(oi.product_name) AS product_name,
		       MAX(oi.sku) AS sku,
		       MAX(oi.variant_label) AS variant_label,
		       p.brand_id,
		       br.name AS brand_name,
		       o.preorder_batch_id AS batch_id,
		       MAX(b.name) AS batch_name,
		       COALESCE(
		         (array_agg(o.campaign_id) FILTER (WHERE o.campaign_id IS NOT NULL))[1],
		         b.campaign_id
		       ) AS campaign_id,
		       COALESCE(SUM(oi.quantity) FILTER (
		         WHERE o.order_status IN ('confirmed','completed')
		           AND o.payment_status='paid'
		       ),0) AS required_qty,
		       COALESCE(SUM(oi.quantity) FILTER (
		         WHERE o.order_status IN ('confirmed','pending_confirmation')
		           AND o.payment_status IN ('unpaid','awaiting_verification')
		       ),0) AS unpaid_demand
		FROM order_items oi
		JOIN orders o ON o.id=oi.order_id AND o.store_id=oi.store_id
		JOIN preorder_batches b ON b.id=o.preorder_batch_id AND b.store_id=o.store_id
		LEFT JOIN products p ON p.id=oi.product_id AND p.store_id=oi.store_id
		LEFT JOIN brands br ON br.id=p.brand_id AND br.store_id=p.store_id
		WHERE ` + whereSQL + `
		GROUP BY oi.product_id, oi.variant_id, p.brand_id, br.name, o.preorder_batch_id, b.campaign_id
		ORDER BY batch_name NULLS LAST, product_name, variant_label NULLS FIRST
	`
	rows, err := s.DB.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []PurchasingLine
	for rows.Next() {
		var line PurchasingLine
		if err := rows.Scan(
			&line.ProductID, &line.VariantID, &line.ProductName, &line.SKU, &line.VariantLabel,
			&line.BrandID, &line.BrandName, &line.BatchID, &line.BatchName, &line.CampaignID,
			&line.RequiredQty, &line.UnpaidDemand,
		); err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]PurchasingLine, 0, len(lines))
	for _, line := range lines {
		batchID := uuid.Nil
		if line.BatchID != nil {
			batchID = *line.BatchID
		}

		// Preferred vendor enrichment (VEN-03); filter when vendor_id set.
		prefVendorID, prefVendorName := s.preferredVendor(ctx, storeID, line.ProductID)
		if f.VendorID != nil {
			if prefVendorID == nil || *prefVendorID != *f.VendorID {
				// Still include if any purchasing record exists for this vendor.
				var hasSplit bool
				_ = s.DB.QueryRow(ctx, `
					SELECT EXISTS(
						SELECT 1 FROM preorder_purchasing_records
						WHERE store_id=$1 AND batch_id=$2 AND product_id=$3
						  AND (($4::uuid IS NULL AND variant_id IS NULL) OR variant_id=$4)
						  AND vendor_id=$5
					)
				`, storeID, batchID, line.ProductID, line.VariantID, *f.VendorID).Scan(&hasSplit)
				if !hasSplit {
					continue
				}
			}
			line.VendorID = f.VendorID
			_ = s.DB.QueryRow(ctx, `SELECT name FROM vendors WHERE id=$1 AND store_id=$2`, *f.VendorID, storeID).Scan(&prefVendorName)
			line.VendorName = prefVendorName
		} else {
			line.VendorID = prefVendorID
			line.VendorName = prefVendorName
		}

		if f.VendorID != nil {
			_ = s.DB.QueryRow(ctx, `
				SELECT COALESCE(SUM(purchased_qty),0), COALESCE(SUM(unavailable_qty),0),
				       COALESCE(SUM(received_qty),0), COALESCE(MAX(version),0)
				FROM preorder_purchasing_records
				WHERE store_id=$1 AND batch_id=$2 AND product_id=$3
				  AND (($4::uuid IS NULL AND variant_id IS NULL) OR variant_id=$4)
				  AND vendor_id=$5
			`, storeID, batchID, line.ProductID, line.VariantID, *f.VendorID).Scan(
				&line.PurchasedQty, &line.UnavailableQty, &line.ReceivedQty, &line.Version,
			)
		} else {
			_ = s.DB.QueryRow(ctx, `
				SELECT COALESCE(SUM(purchased_qty),0), COALESCE(SUM(unavailable_qty),0),
				       COALESCE(SUM(received_qty),0), COALESCE(MAX(version),0)
				FROM preorder_purchasing_records
				WHERE store_id=$1 AND batch_id=$2 AND product_id=$3
				  AND (($4::uuid IS NULL AND variant_id IS NULL) OR variant_id=$4)
			`, storeID, batchID, line.ProductID, line.VariantID).Scan(
				&line.PurchasedQty, &line.UnavailableQty, &line.ReceivedQty, &line.Version,
			)
		}
		_ = s.DB.QueryRow(ctx, `
			SELECT COALESCE(SUM(allocated_qty),0) FROM preorder_item_allocations
			WHERE store_id=$1 AND batch_id=$2 AND product_id=$3
			  AND (($4::uuid IS NULL AND variant_id IS NULL) OR variant_id=$4)
		`, storeID, batchID, line.ProductID, line.VariantID).Scan(&line.AllocatedQty)

		outQty := line.RequiredQty - line.PurchasedQty
		if outQty < 0 {
			outQty = 0
		}
		line.OutstandingQty = outQty
		out = append(out, line)
	}

	if groupBy == "vendor" || groupBy == "brand" || groupBy == "campaign" {
		out = collapsePurchasingLines(out, groupBy)
	}
	return out, nil
}

func (s *Service) preferredVendor(ctx context.Context, storeID, productID uuid.UUID) (*uuid.UUID, *string) {
	var id uuid.UUID
	var name string
	err := s.DB.QueryRow(ctx, `
		SELECT v.id, v.name
		FROM product_vendors pv
		JOIN vendors v ON v.id=pv.vendor_id AND v.store_id=pv.store_id
		WHERE pv.store_id=$1 AND pv.product_id=$2 AND v.status='active'
		ORDER BY pv.is_preferred DESC, v.name
		LIMIT 1
	`, storeID, productID).Scan(&id, &name)
	if err != nil {
		return nil, nil
	}
	return &id, &name
}

func collapsePurchasingLines(lines []PurchasingLine, groupBy string) []PurchasingLine {
	type key struct {
		g string
		p uuid.UUID
		v string
	}
	index := map[key]int{}
	var out []PurchasingLine
	for _, line := range lines {
		var g string
		switch groupBy {
		case "vendor":
			if line.VendorID != nil {
				g = line.VendorID.String()
			} else {
				g = "none"
			}
		case "brand":
			if line.BrandID != nil {
				g = line.BrandID.String()
			} else {
				g = "none"
			}
		case "campaign":
			if line.CampaignID != nil {
				g = line.CampaignID.String()
			} else {
				g = "none"
			}
		default:
			out = append(out, line)
			continue
		}
		vKey := ""
		if line.VariantID != nil {
			vKey = line.VariantID.String()
		}
		k := key{g: g, p: line.ProductID, v: vKey}
		if i, ok := index[k]; ok {
			out[i].RequiredQty += line.RequiredQty
			out[i].UnpaidDemand += line.UnpaidDemand
			out[i].PurchasedQty += line.PurchasedQty
			out[i].UnavailableQty += line.UnavailableQty
			out[i].AllocatedQty += line.AllocatedQty
			out[i].ReceivedQty += line.ReceivedQty
			outQty := out[i].RequiredQty - out[i].PurchasedQty
			if outQty < 0 {
				outQty = 0
			}
			out[i].OutstandingQty = outQty
			continue
		}
		index[k] = len(out)
		out = append(out, line)
	}
	return out
}
