package commerce

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rubicahq/pikko/services/api/internal/auth"
	"github.com/rubicahq/pikko/services/api/internal/email"
	"github.com/rubicahq/pikko/services/api/internal/money"
	"github.com/rubicahq/pikko/services/api/internal/validate"
)

type publicStoreMeta struct {
	ID              uuid.UUID
	Name            string
	Slug            string
	Currency        string
	CheckoutEnabled bool
	Published       bool
	Archived        bool
	OwnerEmail      string
	OwnerName       *string
}

func (s *Service) loadPublicStore(ctx context.Context, slug string) (*publicStoreMeta, error) {
	var m publicStoreMeta
	err := s.DB.QueryRow(ctx, `
		SELECT s.id, s.name, s.slug, s.currency, s.checkout_enabled,
		       s.publication_status='published', s.archived_at IS NOT NULL,
		       COALESCE(u.email, ''), u.display_name
		FROM stores s
		LEFT JOIN store_memberships sm ON sm.store_id=s.id AND sm.role='owner' AND sm.status='active'
		LEFT JOIN users u ON u.id=sm.user_id
		WHERE s.slug=$1
	`, slug).Scan(
		&m.ID, &m.Name, &m.Slug, &m.Currency, &m.CheckoutEnabled,
		&m.Published, &m.Archived, &m.OwnerEmail, &m.OwnerName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if m.Archived || !m.Published {
		return nil, ErrStoreUnavailable
	}
	return &m, nil
}

// resolveCartGroup ensures one selling model / one batch per checkout (PRD Â§20).
func resolveCartGroup(items []CartLineInput) (sellingModel string, batchID *uuid.UUID, err error) {
	if len(items) == 0 {
		return "", nil, ErrInvalidInput
	}
	var hasReady, hasPreorder bool
	var firstBatch *uuid.UUID
	for _, it := range items {
		if it.BatchID != nil {
			hasPreorder = true
			if firstBatch == nil {
				firstBatch = it.BatchID
			} else if *firstBatch != *it.BatchID {
				return "", nil, ErrIncompatibleCart
			}
		} else {
			hasReady = true
		}
	}
	if hasReady && hasPreorder {
		return "", nil, ErrIncompatibleCart
	}
	if hasPreorder {
		return "preorder", firstBatch, nil
	}
	return "ready_stock", nil, nil
}

func (s *Service) priceLines(ctx context.Context, storeID uuid.UUID, items []CartLineInput) ([]PricedLine, int64, error) {
	if len(items) == 0 {
		return nil, 0, ErrInvalidInput
	}
	sellingModel, batchID, err := resolveCartGroup(items)
	if err != nil {
		return nil, 0, err
	}
	if sellingModel == "preorder" && batchID != nil {
		if _, err := s.assertBatchOpenForCheckout(ctx, storeID, *batchID); err != nil {
			return nil, 0, err
		}
	}

	seen := map[string]struct{}{}
	var lines []PricedLine
	var subtotal int64
	for _, it := range items {
		if it.Quantity <= 0 || it.Quantity > 999 {
			return nil, 0, ErrInvalidInput
		}
		key := it.ProductID.String()
		if it.VariantID != nil {
			key += ":" + it.VariantID.String()
		}
		if it.BatchID != nil {
			key += ":b:" + it.BatchID.String()
		}
		if _, ok := seen[key]; ok {
			return nil, 0, ErrInvalidInput
		}
		seen[key] = struct{}{}

		var name, slug, status, availability, invMode string
		var basePrice int64
		var sku *string
		var stock *int
		var readyEligible, preorderEligible bool
		err := s.DB.QueryRow(ctx, `
			SELECT name, slug, sku, base_price_idr, status, availability, inventory_mode, stock_quantity,
			       ready_stock_eligible, preorder_eligible
			FROM products WHERE id=$1 AND store_id=$2
		`, it.ProductID, storeID).Scan(
			&name, &slug, &sku, &basePrice, &status, &availability, &invMode, &stock,
			&readyEligible, &preorderEligible,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, 0, ErrNotFound
		}
		if err != nil {
			return nil, 0, err
		}
		if (status != "published" && status != "unlisted") || availability != "available" {
			return nil, 0, ErrInsufficientStock
		}
		if sellingModel == "ready_stock" && !readyEligible {
			return nil, 0, ErrInvalidInput
		}
		if sellingModel == "preorder" && !preorderEligible {
			return nil, 0, ErrInvalidInput
		}

		unit := basePrice
		var variantLabel *string
		var variantID *uuid.UUID
		if it.VariantID != nil {
			var vAvail string
			var override *int64
			var vStock *int
			var optionKey string
			var vSKU *string
			err = s.DB.QueryRow(ctx, `
				SELECT availability, price_override_idr, stock_quantity, option_key, sku
				FROM product_variants WHERE id=$1 AND product_id=$2 AND store_id=$3
			`, *it.VariantID, it.ProductID, storeID).Scan(&vAvail, &override, &vStock, &optionKey, &vSKU)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, 0, ErrNotFound
			}
			if err != nil {
				return nil, 0, err
			}
			if vAvail != "available" {
				return nil, 0, ErrInsufficientStock
			}
			if override != nil {
				unit = *override
			}
			if optionKey != "" {
				variantLabel = &optionKey
			}
			if vSKU != nil {
				sku = vSKU
			}
			if vStock != nil {
				stock = vStock
			}
			variantID = it.VariantID
		}

		if sellingModel == "preorder" && batchID != nil {
			var batchOverride *int64
			var bpAvail string
			err = s.DB.QueryRow(ctx, `
				SELECT price_override_idr, availability FROM preorder_batch_products
				WHERE store_id=$1 AND batch_id=$2 AND product_id=$3
			`, storeID, *batchID, it.ProductID).Scan(&batchOverride, &bpAvail)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, 0, ErrNotFound
			}
			if err != nil {
				return nil, 0, err
			}
			if bpAvail != "available" {
				return nil, 0, ErrInsufficientStock
			}
			if batchOverride != nil {
				unit = *batchOverride
			}
			if err := s.checkBatchQuota(ctx, storeID, *batchID, it.ProductID, variantID, it.Quantity); err != nil {
				return nil, 0, err
			}
			availLeft := 0
			var limit *int
			_ = s.DB.QueryRow(ctx, `
				SELECT quantity_limit FROM preorder_batch_products
				WHERE store_id=$1 AND batch_id=$2 AND product_id=$3
			`, storeID, *batchID, it.ProductID).Scan(&limit)
			if limit != nil {
				var reserved int
				_ = s.DB.QueryRow(ctx, `
					SELECT COALESCE(SUM(quantity),0) FROM inventory_reservations
					WHERE store_id=$1 AND batch_id=$2 AND product_id=$3 AND status='active'
					  AND (($4::uuid IS NULL AND variant_id IS NULL) OR variant_id=$4)
				`, storeID, *batchID, it.ProductID, variantID).Scan(&reserved)
				availLeft = *limit - reserved
				stock = &availLeft
			}
			invMode = "quota"
		} else if invMode == "finite" {
			reserved := 0
			_ = s.DB.QueryRow(ctx, `
				SELECT COALESCE(SUM(quantity),0) FROM inventory_reservations
				WHERE store_id=$1 AND product_id=$2 AND status='active' AND batch_id IS NULL
				  AND (($3::uuid IS NULL AND variant_id IS NULL) OR variant_id=$3)
			`, storeID, it.ProductID, variantID).Scan(&reserved)
			avail := 0
			if stock != nil {
				avail = *stock - reserved
			}
			if avail < it.Quantity {
				return nil, 0, ErrInsufficientStock
			}
			stock = &avail
		}
		lineTotal := unit * int64(it.Quantity)
		lines = append(lines, PricedLine{
			ProductID: it.ProductID, VariantID: variantID, BatchID: it.BatchID,
			ProductName: name, ProductSlug: slug,
			SKU: sku, VariantLabel: variantLabel, UnitPriceIDR: unit, Quantity: it.Quantity,
			LineTotalIDR: lineTotal, AvailableQty: stock, InventoryMode: invMode,
		})
		subtotal = money.SumIDR(subtotal, lineTotal)
	}
	return lines, subtotal, nil
}

func (s *Service) resolveShipping(ctx context.Context, storeID uuid.UUID, method string) (int64, error) {
	f, err := s.GetFulfillmentSettings(ctx, storeID)
	if err != nil {
		return 0, err
	}
	switch method {
	case FulfillmentShipping:
		if !f.ShippingEnabled {
			return 0, ErrInvalidInput
		}
		return f.ShippingFeeIDR, nil
	case FulfillmentFreeShipping:
		if !f.FreeShippingEnabled {
			return 0, ErrInvalidInput
		}
		return 0, nil
	case FulfillmentPickup:
		if !f.PickupEnabled {
			return 0, ErrInvalidInput
		}
		return 0, nil
	default:
		return 0, ErrInvalidInput
	}
}

func (s *Service) PreviewCheckout(ctx context.Context, slug string, items []CartLineInput, fulfillmentMethod string) (*CheckoutPreview, error) {
	st, err := s.loadPublicStore(ctx, slug)
	if err != nil {
		return nil, err
	}
	sellingModel, batchID, err := resolveCartGroup(items)
	if err != nil {
		return nil, err
	}
	lines, subtotal, err := s.priceLines(ctx, st.ID, items)
	if err != nil {
		return nil, err
	}
	var shipFee int64
	if fulfillmentMethod != "" {
		shipFee, err = s.resolveShipping(ctx, st.ID, fulfillmentMethod)
		if err != nil {
			return nil, err
		}
	}
	itemQty := 0
	for _, l := range lines {
		itemQty += l.Quantity
	}
	eval, evalErr := s.EvaluateCheckoutRules(ctx, st.ID, sellingModel, batchID, subtotal, itemQty, shipFee)
	if evalErr != nil && !errors.Is(evalErr, ErrMinimumNotMet) {
		return nil, evalErr
	}
	if eval == nil {
		eval = &RulesEvaluation{SubtotalIDR: subtotal, ShippingFeeIDR: shipFee, TotalIDR: money.SumIDR(subtotal, shipFee), MinimumOK: true}
	}
	batchName, batchSlug := "", ""
	if batchID != nil {
		if b, err := s.GetBatch(ctx, st.ID, *batchID, false); err == nil {
			batchName, batchSlug = b.Name, b.Slug
		}
	}
	methods, _ := s.ListEnabledPaymentMethods(ctx, st.ID)
	fulfillment, _ := s.GetFulfillmentSettings(ctx, st.ID)
	return &CheckoutPreview{
		StoreID: st.ID, StoreSlug: st.Slug, StoreName: st.Name, Currency: st.Currency,
		SellingModel: sellingModel, BatchID: batchID, BatchName: batchName, BatchSlug: batchSlug,
		Lines: lines, SubtotalIDR: subtotal, ShippingFeeIDR: shipFee, HandlingFeeIDR: eval.HandlingFeeIDR,
		TotalIDR: eval.TotalIDR, RequiresApproval: eval.RequiresApproval,
		MinimumOK: eval.MinimumOK, MinimumMessage: eval.MinimumMessage, AppliedRules: eval.Applied,
		FulfillmentMethod: fulfillmentMethod,
		CheckoutEnabled:   st.CheckoutEnabled, PaymentMethods: methods, Fulfillment: fulfillment,
	}, nil
}

type SubmitResult struct {
	Order         *Order   `json:"order"`
	Payment       *Payment `json:"payment"`
	TrackingToken string   `json:"tracking_token"`
	TrackingURL   string   `json:"tracking_url"`
	CheckoutURL   ~5ãO-¢G§²ÚîÆ­yÝŒ¹MÑ½É•%¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•ÑÕÉ¸µ…ÁmÍÑÉ¥¹u…¹åì‰¡•­±¥ÍÐˆè±ô°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤•ÑMÑ½É•™É½¹ÑMÑ…ÑÕÌ¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°|©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%‘É…™Ð°•ÉÈ€èôÌ¹MÑ½É•™É½¹Ð¹•Ð¡Ñà°Œ¹MÑ½É•%°€‰‘É…™Ðˆ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%ÁÕ‰±¥Í¡•°•ÉÈ€èôÌ¹MÑ½É•™É½¹Ð¹•Ð¡Ñà°Œ¹MÑ½É•%°€‰ÁÕ‰±¥Í¡•ˆ¤(%½ÕÐ€èôµ…ÁmÍÑÉ¥¹u…¹åì‰‘É…™Ðˆè‘É…™Ð°€‰¡…Í}ÁÕ‰±¥Í¡•ˆè•ÉÈ€ôô¹¥±ô(%¥˜•ÉÈ€ôô¹¥°ì($%½ÕÑl‰ÁÕ‰±¥Í¡•‰t€ôÁÕ‰±¥Í¡•(%ô(%É•ÑÕÉ¸½ÕÐ°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤±¥ÍÑAÉ½‘ÕÑÌ¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°…ÉÌ©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%¥¸°•ÉÈ€èô‘•½‘•ÉÍmÍÑÉÕÐì($%EÕ•Éä€ÍÑÉ¥¹œ©Í½¸è‰ÅÕ•Éä‰€($%MÑ…ÑÕÌÍÑÉ¥¹œ©Í½¸è‰ÍÑ…ÑÕÌ‰€($%1¥µ¥Ð€¥¹Ð€€€©Í½¸è‰±¥µ¥Ð‰€($%=™™Í•Ð¥¹Ð€€€©Í½¸è‰½™™Í•Ð‰€(%õt¡…ÉÌ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%±¥ÍÐ°Ñ½Ñ…°°•ÉÈ€èôÌ¹…Ñ…±½œ¹1¥ÍÑAÉ½‘ÕÑÌ¡Ñà°Œ¹MÑ½É•%°…Ñ…±½œ¹AÉ½‘ÕÑ1¥ÍÑ¥±Ñ•Éì($%EÕ•Éäè€¥¸¹EÕ•Éä°($%MÑ…ÑÕÌè¥¸¹MÑ…ÑÕÌ°($%1¥µ¥Ðè€±…µÁ1¥µ¥Ð¡¥¸¹1¥µ¥Ð°€ÈÔ°€ÄÀÀ¤°($%=™™Í•Ðè¥¸¹=™™Í•Ð°(%ô°™…±Í”¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•ÑÕÉ¸µ…ÁmÍÑÉ¥¹u…¹åì‰ÁÉ½‘ÕÑÌˆè±¥ÍÐ°€‰Ñ½Ñ…°ˆèÑ½Ñ…±ô°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤•ÑAÉ½‘ÕÐ¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°…ÉÌ©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%¥¸°•ÉÈ€èô‘•½‘•ÉÍmÍÑÉÕÐì($%AÉ½‘ÕÑ%ÍÑÉ¥¹œ©Í½¸è‰ÁÉ½‘ÕÑ}¥‰€(%õt¡…ÉÌ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%¥°•ÉÈ€èôÁ…ÉÍ•UU%Éœ¡¥¸¹AÉ½‘ÕÑ%°€‰ÁÉ½‘ÕÑ}¥ˆ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%À°•ÉÈ€èôÌ¹…Ñ…±½œ¹•ÑAÉ½‘ÕÐ¡Ñà°Œ¹MÑ½É•%°¥°™…±Í”¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•ÑÕÉ¸µ…ÁmÍÑÉ¥¹u…¹åì‰ÁÉ½‘ÕÐˆèÁô°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤±¥ÍÑ…Ñ…±½Ì¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°|©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%±¥ÍÐ°•ÉÈ€èôÌ¹…Ñ…±½Ì¹1¥ÍÐ¡Ñà°Œ¹MÑ½É•%¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•ÑÕÉ¸µ…ÁmÍÑÉ¥¹u…¹åì‰…Ñ…±½Ìˆè±¥ÍÑô°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤•Ñ…Ñ…±½œ¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°…ÉÌ©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%¥¸°•ÉÈ€èô‘•½‘•ÉÍmÍÑÉÕÐì($%…Ñ…±½%ÍÑÉ¥¹œ©Í½¸è‰…Ñ…±½}¥‰€(%õt¡…ÉÌ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%¥°•ÉÈ€èôÁ…ÉÍ•UU%Éœ¡¥¸¹…Ñ…±½%°€‰…Ñ…±½}¥ˆ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%Œ°•ÉÈ€èôÌ¹…Ñ…±½Ì¹•Ð¡Ñà°Œ¹MÑ½É•%°¥¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•ÑÕÉ¸µ…ÁmÍÑÉ¥¹u…¹åì‰…Ñ…±½œˆèô°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤±¥ÍÑ…µÁ…¥¹Ì¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°…ÉÌ©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%¥¸°•ÉÈ€èô‘•½‘•ÉÍmÍÑÉÕÐì($%QåÁ”ÍÑÉ¥¹œ©Í½¸è‰ÑåÁ”‰€(%õt¡…ÉÌ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%±¥ÍÐ°•ÉÈ€èôÌ¹…µÁ…¥¹Ì¹1¥ÍÐ¡Ñà°Œ¹MÑ½É•%°¥¸¹QåÁ”¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•ÑÕÉ¸µ…ÁmÍÑÉ¥¹u…¹åì‰…µÁ…¥¹Ìˆè±¥ÍÑô°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤•Ñ…µÁ…¥¸¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°…ÉÌ©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%¥¸°•ÉÈ€èô‘•½‘•ÉÍmÍÑÉÕÐì($%…µÁ…¥¹%ÍÑÉ¥¹œ©Í½¸è‰…µÁ…¥¹}¥‰€(%õt¡…ÉÌ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%¥°•ÉÈ€èôÁ…ÉÍ•UU%Éœ¡¥¸¹…µÁ…¥¹%°€‰…µÁ…¥¹}¥ˆ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%Œ°•ÉÈ€èôÌ¹…µÁ…¥¹Ì¹•Ð¡Ñà°Œ¹MÑ½É•%°¥°ÑÉÕ”¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•ÑÕÉ¸µ…ÁmÍÑÉ¥¹u…¹åì‰…µÁ…¥¸ˆèô°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤±¥ÍÑ=É‘•ÉÌ¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°…ÉÌ©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%¥¸°•ÉÈ€èô‘•½‘•ÉÍmÍÑÉÕÐì($%EÕ•Éä€€€€€€€€€€€€ÍÑÉ¥¹œ©Í½¸è‰ÅÕ•Éä‰€($%=É‘•ÉMÑ…ÑÕÌ€€€€€€ÍÑÉ¥¹œ©Í½¸è‰½É‘•É}ÍÑ…ÑÕÌ‰€($%A…åµ•¹ÑMÑ…ÑÕÌ€€€€ÍÑÉ¥¹œ©Í½¸è‰Á…åµ•¹Ñ}ÍÑ…ÑÕÌ‰€($%Õ±™¥±±µ•¹ÑMÑ…ÑÕÌÍÑÉ¥¹œ©Í½¸è‰™Õ±™¥±±µ•¹Ñ}ÍÑ…ÑÕÌ‰€($%1¥µ¥Ð€€€€€€€€€€€€¥¹Ð€€€©Í½¸è‰±¥µ¥Ð‰€($%=™™Í•Ð€€€€€€€€€€€¥¹Ð€€€©Í½¸è‰½™™Í•Ð‰€(%õt¡…ÉÌ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%±¥ÍÐ°Ñ½Ñ…°°•ÉÈ€èôÌ¹½µµ•É”¹1¥ÍÑ=É‘•ÉÌ¡Ñà°Œ¹MÑ½É•%°½µµ•É”¹1¥ÍÑ=É‘•ÉÍ¥±Ñ•Éì($%EÕ•Éäè€€€€€€€€€€€€¥¸¹EÕ•Éä°($%=É‘•ÉMÑ…ÑÕÌè€€€€€€¥¸¹=É‘•ÉMÑ…ÑÕÌ°($%A…åµ•¹ÑMÑ…ÑÕÌè€€€€¥¸¹A…åµ•¹ÑMÑ…ÑÕÌ°($%Õ±™¥±±µ•¹ÑMÑ…ÑÕÌè¥¸¹Õ±™¥±±µ•¹ÑMÑ…ÑÕÌ°($%1¥µ¥Ðè€€€€€€€€€€€€±…µÁ1¥µ¥Ð¡¥¸¹1¥µ¥Ð°€ÈÔ°€ÄÀÀ¤°($%=™™Í•Ðè€€€€€€€€€€€¥¸¹=™™Í•Ð°(%ô¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•ÑÕÉ¸µ…ÁmÍÑÉ¥¹u…¹åì‰½É‘•ÉÌˆè±¥ÍÐ°€‰Ñ½Ñ…°ˆèÑ½Ñ…±ô°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤•Ñ=É‘•È¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°…ÉÌ©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%¥¸°•ÉÈ€èô‘•½‘•ÉÍmÍÑÉÕÐì($%=É‘•É%ÍÑÉ¥¹œ©Í½¸è‰½É‘•É}¥‰€(%õt¡…ÉÌ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%¥°•ÉÈ€èôÁ…ÉÍ•UU%Éœ¡¥¸¹=É‘•É%°€‰½É‘•É}¥ˆ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%¼°•ÉÈ€èôÌ¹½µµ•É”¹•Ñ=É‘•È¡Ñà°Œ¹MÑ½É•%°¥¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•ÑÕÉ¸µ…ÁmÍÑÉ¥¹u…¹åì‰½É‘•Èˆè½ô°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤±¥ÍÑ=É‘•ÉA…åµ•¹ÑÌ¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°…ÉÌ©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%¥¸°•ÉÈ€èô‘•½‘•ÉÍmÍÑÉÕÐì($%=É‘•É%ÍÑÉ¥¹œ©Í½¸è‰½É‘•É}¥‰€(%õt¡…ÉÌ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%¥°•ÉÈ€èôÁ…ÉÍ•UU%Éœ¡¥¸¹=É‘•É%°€‰½É‘•É}¥ˆ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%±¥ÍÐ°•ÉÈ€èôÌ¹½µµ•É”¹1¥ÍÑ=É‘•ÉA…åµ•¹ÑÌ¡Ñà°Œ¹MÑ½É•%°¥¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•ÑÕÉ¸µ…ÁmÍÑÉ¥¹u…¹åì‰Á…åµ•¹ÑÌˆè±¥ÍÑô°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤±¥ÍÑÕÍÑ½µ•ÉÌ¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°…ÉÌ©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%¥¸°•ÉÈ€èô‘•½‘•ÉÍmÍÑÉÕÐì($%EÕ•Éä€ÍÑÉ¥¹œ©Í½¸è‰ÅÕ•Éä‰€($%1¥µ¥Ð€¥¹Ð€€€©Í½¸è‰±¥µ¥Ð‰€($%=™™Í•Ð¥¹Ð€€€©Í½¸è‰½™™Í•Ð‰€(%õt¡…ÉÌ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%±¥ÍÐ°Ñ½Ñ…°°•ÉÈ€èôÌ¹½µµ•É”¹1¥ÍÑÕÍÑ½µ•ÉÌ¡Ñà°Œ¹MÑ½É•%°¥¸¹EÕ•Éä°±…µÁ1¥µ¥Ð¡¥¸¹1¥µ¥Ð°€ÈÔ°€ÄÀÀ¤°¥¸¹=™™Í•Ð¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•ÑÕÉ¸µ…ÁmÍÑÉ¥¹u…¹åì‰ÕÍÑ½µ•ÉÌˆè±¥ÍÐ°€‰Ñ½Ñ…°ˆèÑ½Ñ…±ô°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤•ÑÕÍÑ½µ•È¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°…ÉÌ©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%¥¸°•ÉÈ€èô‘•½‘•ÉÍmÍÑÉÕÐì($%ÕÍÑ½µ•É%ÍÑÉ¥¹œ©Í½¸è‰ÕÍÑ½µ•É}¥‰€(%õt¡…ÉÌ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%¥°•ÉÈ€èôÁ…ÉÍ•UU%Éœ¡¥¸¹ÕÍÑ½µ•É%°€‰ÕÍÑ½µ•É}¥ˆ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%Œ°•ÉÈ€èôÌ¹½µµ•É”¹•ÑÕÍÑ½µ•È¡Ñà°Œ¹MÑ½É•%°¥¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•ÑÕÉ¸µ…ÁmÍÑÉ¥¹u…¹åì‰ÕÍÑ½µ•Èˆèô°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤É•…Ñ•ÕÍÑ½µ•È¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°…ÉÌ©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%¥¸°•ÉÈ€èô‘•½‘•ÉÍm½µµ•É”¹É•…Ñ•ÕÍÑ½µ•É%¹ÁÕÑt¡…ÉÌ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•Ì°•ÉÈ€èôÌ¹½µµ•É”¹É•…Ñ•ÕÍÑ½µ•È¡Ñà°Œ¹MÑ½É•%°¥¸¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•ÑÕÉ¸É•Ì°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤±¥ÍÑAÉ•½É‘•ÉÌ¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°|©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%±¥ÍÐ°•ÉÈ€èôÌ¹½µµ•É”¹1¥ÍÑ	…Ñ¡•Ì¡Ñà°Œ¹MÑ½É•%¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•ÑÕÉ¸µ…ÁmÍÑÉ¥¹u…¹åì‰ÁÉ•½É‘•ÉÌˆè±¥ÍÑô°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤•ÑAÉ•½É‘•È¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°…ÉÌ©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%¥¸°•ÉÈ€èô‘•½‘•ÉÍmÍÑÉÕÐì($%	…Ñ¡%ÍÑÉ¥¹œ©Í½¸è‰‰…Ñ¡}¥‰€(%õt¡…ÉÌ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%¥°•ÉÈ€èôÁ…ÉÍ•UU%Éœ¡¥¸¹	…Ñ¡%°€‰‰…Ñ¡}¥ˆ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%ˆ°•ÉÈ€èôÌ¹½µµ•É”¹•Ñ	…Ñ ¡Ñà°Œ¹MÑ½É•%°¥°ÑÉÕ”¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•ÑÕÉ¸µ…ÁmÍÑÉ¥¹u…¹åì‰ÁÉ•½É‘•Èˆè‰ô°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤•ÑAÕÉ¡…Í¥¹MÕµµ…Éä¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°…ÉÌ©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%¥¸°•ÉÈ€èô‘•½‘•ÉÍmÍÑÉÕÐì($%	…Ñ¡%€€€ÍÑÉ¥¹œ©Í½¸è‰‰…Ñ¡}¥‰€($%…µÁ…¥¹%ÍÑÉ¥¹œ©Í½¸è‰…µÁ…¥¹}¥‰€($%É½ÕÁ	ä€€€ÍÑÉ¥¹œ©Í½¸è‰É½ÕÁ}‰ä‰€(%õt¡…ÉÌ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%˜€èô½µµ•É”¹AÕÉ¡…Í¥¹¥±Ñ•ÉíÉ½ÕÁ	äè¥¸¹É½ÕÁ	åô(%¥˜¥¸¹	…Ñ¡%€„ô€ˆˆì($%¥°•ÉÈ€èôÁ…ÉÍ•UU%Éœ¡¥¸¹	…Ñ¡%°€‰‰…Ñ¡}¥ˆ¤($%¥˜•ÉÈ€„ô¹¥°ì($$%É•ÑÕÉ¸¹¥°°•ÉÈ($%ô($%˜¹	…Ñ¡%€ô€™¥(%ô(%¥˜¥¸¹…µÁ…¥¹%€„ô€ˆˆì($%¥°•ÉÈ€èôÁ…ÉÍ•UU%Éœ¡¥¸¹…µÁ…¥¹%°€‰…µÁ…¥¹}¥ˆ¤($%¥˜•ÉÈ€„ô¹¥°ì($$%É•ÑÕÉ¸¹¥°°•ÉÈ($%ô($%˜¹…µÁ…¥¹%€ô€™¥(%ô(%±¥ÍÐ°•ÉÈ€èôÌ¹½µµ•É”¹É•…Ñ•AÕÉ¡…Í¥¹œ¡Ñà°Œ¹MÑ½É•%°˜¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•ÑÕÉ¸µ…ÁmÍÑÉ¥¹u…¹åì‰±¥¹•Ìˆè±¥ÍÑô°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤•ÑA…­¥¹EÕ•Õ”¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°…ÉÌ©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%¥¸°•ÉÈ€èô‘•½‘•ÉÍmÍÑÉÕÐì($%É½ÕÁ	äÍÑÉ¥¹œ©Í½¸è‰É½ÕÁ}‰ä‰€($%MÑ…ÑÕÌ€ÍÑÉ¥¹œ©Í½¸è‰ÍÑ…ÑÕÌ‰€(%õt¡…ÉÌ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%½ÕÐ°•ÉÈ€èôÌ¹½µµ•É”¹1¥ÍÑA…­¥¹EÕ•Õ”¡Ñà°Œ¹MÑ½É•%°¥¸¹É½ÕÁ	ä°¥¸¹MÑ…ÑÕÌ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•ÑÕÉ¸½ÕÐ°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤±¥ÍÑM¡¥Áµ•¹ÑÌ¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°…ÉÌ©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%¥¸°•ÉÈ€èô‘•½‘•ÉÍmÍÑÉÕÐì($%=É‘•É%ÍÑÉ¥¹œ©Í½¸è‰½É‘•É}¥‰€($%1¥µ¥Ð€€¥¹Ð€€€©Í½¸è‰±¥µ¥Ð‰€($%=™™Í•Ð€¥¹Ð€€€©Í½¸è‰½™™Í•Ð‰€(%õt¡…ÉÌ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%Ù…È½É‘•É%€©ÕÕ¥¹UU%(%¥˜¥¸¹=É‘•É%€„ô€ˆˆì($%¥°•ÉÈ€èôÁ…ÉÍ•UU%Éœ¡¥¸¹=É‘•É%°€‰½É‘•É}¥ˆ¤($%¥˜•ÉÈ€„ô¹¥°ì($$%É•ÑÕÉ¸¹¥°°•ÉÈ($%ô($%½É‘•É%€ô€™¥(%ô(%±¥ÍÐ°Ñ½Ñ…°°•ÉÈ€èôÌ¹½µµ•É”¹1¥ÍÑM¡¥Áµ•¹ÑÌ¡Ñà°Œ¹MÑ½É•%°½É‘•É%°±…µÁ1¥µ¥Ð¡¥¸¹1¥µ¥Ð°€ÈÔ°€ÄÀÀ¤°¥¸¹=™™Í•Ð¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•ÑÕÉ¸µ…ÁmÍÑÉ¥¹u…¹åì‰Í¡¥Áµ•¹ÑÌˆè±¥ÍÐ°€‰Ñ½Ñ…°ˆèÑ½Ñ…±ô°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤•Ñ¥¹…¹•]½É­ÍÁ…”¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°…ÉÌ©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%¥¸°•ÉÈ€èô‘•½‘•ÉÍmÍÑÉÕÐì($%M•ÑÑ±•µ•¹ÑMÑ…ÑÕÌÍÑÉ¥¹œ©Í½¸è‰Í•ÑÑ±•µ•¹Ñ}ÍÑ…ÑÕÌ‰€($%1¥µ¥Ð€€€€€€€€€€€¥¹Ð€€€©Í½¸è‰±¥µ¥Ð‰€($%=™™Í•Ð€€€€€€€€€€¥¹Ð€€€©Í½¸è‰½™™Í•Ð‰€(%õt¡…ÉÌ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%±¥ÍÐ°Ñ½Ñ…°°•ÉÈ€èôÌ¹½µµ•É”¹1¥ÍÑ¥¹…¹•]½É­ÍÁ…”¡Ñà°Œ¹MÑ½É•%°¥¸¹M•ÑÑ±•µ•¹ÑMÑ…ÑÕÌ°±…µÁ1¥µ¥Ð¡¥¸¹1¥µ¥Ð°€ÈÔ°€ÄÀÀ¤°¥¸¹=™™Í•Ð¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•ÑÕÉ¸µ…ÁmÍÑÉ¥¹u…¹åì‰½É‘•ÉÌˆè±¥ÍÐ°€‰Ñ½Ñ…°ˆèÑ½Ñ…±ô°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤•Ñ=É‘•É¥¹…¹”¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°…ÉÌ©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%¥¸°•ÉÈ€èô‘•½‘•ÉÍmÍÑÉÕÐì($%=É‘•É%ÍÑÉ¥¹œ©Í½¸è‰½É‘•É}¥‰€(%õt¡…ÉÌ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%¥°•ÉÈ€èôÁ…ÉÍ•UU%Éœ¡¥¸¹=É‘•É%°€‰½É‘•É}¥ˆ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%ÍÕ´°•ÉÈ€èôÌ¹½µµ•É”¹•Ñ=É‘•É¥¹…¹”¡Ñà°Œ¹MÑ½É•%°¥¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•ÑÕÉ¸µ…ÁmÍÑÉ¥¹u…¹åì‰™¥¹…¹”ˆèÍÕµô°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤•ÑMÕ‰ÍÉ¥ÁÑ¥½¸¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°|©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%Í¹…À°•ÉÈ€èôÌ¹¹Ð¹M¹…ÁÍ¡½Ð¡Ñà°Œ¹MÑ½É•%¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•ÑÕÉ¸Í¹…À°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤•ÑMÕ‰ÍÉ¥ÁÑ¥½¹UÍ…”¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°|©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%Í¹…À°•ÉÈ€èôÌ¹¹Ð¹M¹…ÁÍ¡½Ð¡Ñà°Œ¹MÑ½É•%¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%Á•É¥½€èô€ˆˆ(%¥˜±•¸¡Í¹…À¹UÍ…”¤€ø€Àì($%Á•É¥½€ôÍ¹…À¹UÍ…•lÁt¹A•É¥½‘e4(%ô(%É•ÑÕÉ¸µ…ÁmÍÑÉ¥¹u…¹åì($$‰ÕÍ…”ˆè€Í¹…À¹UÍ…”°($$‰±¥µ¥ÑÌˆèÍ¹…À¹1¥µ¥ÑÌ°($$‰Á•É¥½ˆèÁ•É¥½°(%ô°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤•Ñ¹…±åÑ¥Í=Ù•ÉÙ¥•Ü¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°…ÉÌ©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%¥¸°•ÉÈ€èô‘•½‘•ÉÍmÍÑÉÕÐì($%É½´ÍÑÉ¥¹œ©Í½¸è‰™É½´‰€($%Q¼€€ÍÑÉ¥¹œ©Í½¸è‰Ñ¼‰€(%õt¡…ÉÌ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%Ñè€èô€‰Í¥„½)…­…ÉÑ„ˆ(%¥˜Ì¹™œ€„ô¹¥°€˜˜Ì¹™œ¹•™…Õ±ÑQh€„ô€ˆˆì($%Ñè€ôÌ¹™œ¹•™…Õ±ÑQh(%ô(%½Ø°•ÉÈ€èôÌ¹¹…±åÑ¥Ì¹=Ù•ÉÙ¥•Ü¡Ñà°Œ¹MÑ½É•%°¥¸¹É½´°¥¸¹Q¼°Ñè¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•ÑÕÉ¸µ…ÁmÍÑÉ¥¹u…¹åì‰…¹…±åÑ¥Ìˆè½Ùô°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤±¥ÍÑ9½Ñ¥™¥…Ñ¥½¹Ì¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°…ÉÌ©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%¥¸°•ÉÈ€èô‘•½‘•ÉÍmÍÑÉÕÐì($%U¹É•…‘=¹±ä‰½½°©Í½¸è‰Õ¹É•…‘}½¹±ä‰€($%1¥µ¥Ð€€€€€¥¹Ð€©Í½¸è‰±¥µ¥Ð‰€($%=™™Í•Ð€€€€¥¹Ð€©Í½¸è‰½™™Í•Ð‰€(%õt¡…ÉÌ¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%±¥ÍÐ°Ñ½Ñ…°°•ÉÈ€èôÌ¹9½Ñ¥™¥…Ñ¥½¹Ì¹1¥ÍÐ¡Ñà°Œ¹MÑ½É•%°¥¸¹U¹É•…‘=¹±ä°±…µÁ1¥µ¥Ð¡¥¸¹1¥µ¥Ð°€ÈÀ°€ÄÀÀ¤°¥¸¹=™™Í•Ð¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•ÑÕÉ¸µ…ÁmÍÑÉ¥¹u…¹åì‰¹½Ñ¥™¥…Ñ¥½¹Ìˆè±¥ÍÐ°€‰Ñ½Ñ…°ˆèÑ½Ñ…±ô°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤•ÑÕ±™¥±±µ•¹ÑM•ÑÑ¥¹Ì¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°|©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%™Ì°•ÉÈ€èôÌ¹½µµ•É”¹•ÑÕ±™¥±±µ•¹ÑM•ÑÑ¥¹Ì¡Ñà°Œ¹MÑ½É•%¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•ÑÕÉ¸µ…ÁmÍÑÉ¥¹u…¹åì‰™Õ±™¥±±µ•¹Ðˆè™Íô°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤±¥ÍÑA…åµ•¹Ñ5•Ñ¡½‘Ì¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°|©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%±¥ÍÐ°•ÉÈ€èôÌ¹½µµ•É”¹1¥ÍÑA…åµ•¹Ñ5•Ñ¡½‘Ì¡Ñà°Œ¹MÑ½É•%¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•ÑÕÉ¸µ…ÁmÍÑÉ¥¹u…¹åì‰Á…åµ•¹Ñ}µ•Ñ¡½‘Ìˆè±¥ÍÑô°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤±¥ÍÑQ•…µ5•µ‰•ÉÌ¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°|©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%±¥ÍÐ°•ÉÈ€èôÌ¹Q•…´¹1¥ÍÑ5•µ‰•ÉÌ¡Ñà°Œ¹MÑ½É•%¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•ÑÕÉ¸µ…ÁmÍÑÉ¥¹u…¹åì‰µ•µ‰•ÉÌˆè±¥ÍÑô°¹¥°)ô()™Õ¹Œ€¡Ì€©M•ÉÙ¥”¤±¥ÍÑY•¹‘½ÉÌ¡Ñà½¹Ñ•áÐ¹½¹Ñ•áÐ°Œ…±±½¹Ñ•áÐ°|©Í½¸¹I…Ý5•ÍÍ…”¤€¡…¹ä°•ÉÉ½È¤ì(%±¥ÍÐ°•ÉÈ€èôÌ¹…Ñ…±½œ¹1¥ÍÑY•¹‘½ÉÌ¡Ñà°Œ¹MÑ½É•%°™…±Í”¤(%¥˜•ÉÈ€„ô¹¥°ì($%É•ÑÕÉ¸¹¥°°•ÉÈ(%ô(%É•ÑÕÉ¸µ…ÁmÍÑÉ¥¹u…¹åì‰Ù•¹‘½ÉÌˆè±¥ÍÑô°¹¥°)ô(