// Package whatsapp provides the merchant WhatsApp template registry and deep-link helpers (PRD §23).
// MVP uses wa.me deep links + copy — no Meta Business API.
package whatsapp

// Audience is who the template is written for.
type Audience string

const (
	AudienceVendor   Audience = "vendor"
	AudienceCustomer Audience = "customer"
)

// Template is a catalog entry with Mustache-like {{var}} placeholders.
type Template struct {
	Code        string   `json:"code"`
	Audience    Audience `json:"audience"`
	Category    string   `json:"category"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Body        string   `json:"body"`
	Variables   []string `json:"variables"`
	Surfaces    []string `json:"surfaces"`
}

// All returns the full WA-V* and WA-C* catalog in stable order.
func All() []Template {
	out := make([]Template, 0, len(vendorTemplates)+len(customerTemplates))
	out = append(out, vendorTemplates...)
	out = append(out, customerTemplates...)
	return out
}

// ByCode looks up a template by code (e.g. WA-C12).
func ByCode(code string) (Template, bool) {
	for _, t := range All() {
		if t.Code == code {
			return t, true
		}
	}
	return Template{}, false
}

// List filters by audience ("vendor"|"customer"|"").
func List(audience string) []Template {
	all := All()
	if audience == "" {
		return all
	}
	out := make([]Template, 0, len(all))
	for _, t := range all {
		if string(t.Audience) == audience {
			out = append(out, t)
		}
	}
	return out
}

var vendorTemplates = []Template{
	{
		Code: "WA-V01", Audience: AudienceVendor, Category: "purchasing",
		Title: "Cek stok / ketersediaan",
		Description: "Tanya vendor apakah barang masih tersedia sebelum order.",
		Body: "Halo {{vendor_contact_name}}, saya dari {{store_name}}.\nMohon konfirmasi ketersediaan:\n{{item_summary}}\nTerima kasih!",
		Variables: []string{"vendor_contact_name", "store_name", "item_summary"},
		Surfaces:  []string{"purchasing", "vendors"},
	},
	{
		Code: "WA-V02", Audience: AudienceVendor, Category: "purchasing",
		Title: "Kirim daftar belanja (PO)",
		Description: "Kirim ringkasan purchase order ke vendor.",
		Body: "Halo {{vendor_contact_name}}, berikut daftar belanja dari {{store_name}} ({{batch_name}}):\n{{item_summary}}\nEstimasi total: {{amount_formatted}}\nMohon konfirmasi bisa diproses ya. Terima kasih!",
		Variables: []string{"vendor_contact_name", "store_name", "batch_name", "item_summary", "amount_formatted"},
		Surfaces:  []string{"purchasing", "vendors"},
	},
	{
		Code: "WA-V03", Audience: AudienceVendor, Category: "purchasing",
		Title: "Konfirmasi harga",
		Description: "Minta konfirmasi harga satuan sebelum bayar.",
		Body: "Halo {{vendor_contact_name}}, dari {{store_name}}.\nMohon konfirmasi harga untuk:\n{{item_summary}}\nApakah masih sesuai? Terima kasih.",
		Variables: []string{"vendor_contact_name", "store_name", "item_summary"},
		Surfaces:  []string{"purchasing", "vendors"},
	},
	{
		Code: "WA-V04", Audience: AudienceVendor, Category: "purchasing",
		Title: "Konfirmasi pembayaran ke vendor",
		Description: "Beritahu vendor bahwa transfer sudah dikirim.",
		Body: "Halo {{vendor_contact_name}}, {{store_name}} sudah transfer {{amount_formatted}} untuk order:\n{{item_summary}}\nMohon dicek dan dikonfirmasi ya. Terima kasih!",
		Variables: []string{"vendor_contact_name", "store_name", "amount_formatted", "item_summary"},
		Surfaces:  []string{"purchasing", "vendors"},
	},
	{
		Code: "WA-V05", Audience: AudienceVendor, Category: "receiving",
		Title: "Follow-up pengiriman vendor",
		Description: "Tanya status pengiriman barang dari vendor.",
		Body: "Halo {{vendor_contact_name}}, saya dari {{store_name}}.\nMau follow-up status pengiriman untuk:\n{{item_summary}}\nKira-kira kapan bisa dikirim / resinya apa? Terima kasih.",
		Variables: []string{"vendor_contact_name", "store_name", "item_summary"},
		Surfaces:  []string{"purchasing", "receiving", "vendors"},
	},
	{
		Code: "WA-V06", Audience: AudienceVendor, Category: "receiving",
		Title: "Konfirmasi barang diterima",
		Description: "Konfirmasi ke vendor bahwa barang sudah sampai.",
		Body: "Halo {{vendor_contact_name}}, barang dari {{store_name}} sudah diterima:\n{{item_summary}}\nTerima kasih atas bantuannya!",
		Variables: []string{"vendor_contact_name", "store_name", "item_summary"},
		Surfaces:  []string{"receiving", "vendors"},
	},
	{
		Code: "WA-V07", Audience: AudienceVendor, Category: "purchasing",
		Title: "Barang tidak tersedia / kosong",
		Description: "Konfirmasi item yang kosong dan minta alternatif.",
		Body: "Halo {{vendor_contact_name}}, dari {{store_name}}.\nUntuk item berikut ternyata kosong / tidak tersedia:\n{{item_summary}}\nApakah ada alternatif atau restock schedule? Terima kasih.",
		Variables: []string{"vendor_contact_name", "store_name", "item_summary"},
		Surfaces:  []string{"purchasing", "receiving", "vendors"},
	},
	{
		Code: "WA-V08", Audience: AudienceVendor, Category: "purchasing",
		Title: "Ubah / batalkan sebagian PO",
		Description: "Minta vendor menyesuaikan atau membatalkan sebagian pesanan.",
		Body: "Halo {{vendor_contact_name}}, dari {{store_name}}.\nMohon bantu sesuaikan / batalkan sebagian order:\n{{item_summary}}\nMohon konfirmasinya. Terima kasih!",
		Variables: []string{"vendor_contact_name", "store_name", "item_summary"},
		Surfaces:  []string{"purchasing", "vendors"},
	},
}

var customerTemplates = []Template{
	{
		Code: "WA-C01", Audience: AudienceCustomer, Category: "orders",
		Title: "Order diterima",
		Description: "Konfirmasi awal bahwa pesanan sudah masuk.",
		Body: "Halo {{customer_name}}! Order {{order_number}} di {{store_name}} sudah kami terima.\nTotal: {{amount_formatted}}\nLacak & upload bukti bayar: {{tracking_url}}",
		Variables: []string{"customer_name", "order_number", "store_name", "amount_formatted", "tracking_url"},
		Surfaces:  []string{"orders", "messages"},
	},
	{
		Code: "WA-C02", Audience: AudienceCustomer, Category: "orders",
		Title: "Order dikonfirmasi",
		Description: "Beritahu customer order sudah dikonfirmasi merchant.",
		Body: "Halo {{customer_name}}, order {{order_number}} di {{store_name}} sudah dikonfirmasi.\nTotal: {{amount_formatted}}\nDetail: {{tracking_url}}",
		Variables: []string{"customer_name", "order_number", "store_name", "amount_formatted", "tracking_url"},
		Surfaces:  []string{"orders", "messages"},
	},
	{
		Code: "WA-C03", Audience: AudienceCustomer, Category: "payment",
		Title: "Pengingat pembayaran (deposit)",
		Description: "Ingatkan deposit belum dibayar.",
		Body: "Halo {{customer_name}}, pengingat deposit untuk order {{order_number}} di {{store_name}}.\nJumlah: {{amount_formatted}}\nUpload bukti: {{tracking_url}}\nTerima kasih!",
		Variables: []string{"customer_name", "order_number", "store_name", "amount_formatted", "tracking_url"},
		Surfaces:  []string{"orders", "finance", "messages"},
	},
	{
		Code: "WA-C04", Audience: AudienceCustomer, Category: "payment",
		Title: "Pengingat pelunasan",
		Description: "Ingatkan sisa tagihan / final payment.",
		Body: "Halo {{customer_name}}, sisa pembayaran order {{order_number}} di {{store_name}} sebesar {{amount_formatted}} sudah bisa dilunasi.\nDetail & upload bukti: {{tracking_url}}",
		Variables: []string{"customer_name", "order_number", "store_name", "amount_formatted", "tracking_url"},
		Surfaces:  []string{"orders", "finance", "messages"},
	},
	{
		Code: "WA-C05", Audience: AudienceCustomer, Category: "payment",
		Title: "Pembayaran terverifikasi",
		Description: "Konfirmasi bukti transfer sudah diterima.",
		Body: "Halo {{customer_name}}, pembayaran order {{order_number}} di {{store_name}} ({{amount_formatted}}) sudah kami verifikasi. Terima kasih!\n{{tracking_url}}",
		Variables: []string{"customer_name", "order_number", "store_name", "amount_formatted", "tracking_url"},
		Surfaces:  []string{"orders", "finance", "messages"},
	},
	{
		Code: "WA-C06", Audience: AudienceCustomer, Category: "payment",
		Title: "Bukti bayar perlu diperbaiki",
		Description: "Minta customer unggah ulang bukti transfer.",
		Body: "Halo {{customer_name}}, bukti bayar order {{order_number}} di {{store_name}} belum bisa kami terima.\nAlasan: {{reason}}\nMohon upload ulang di: {{tracking_url}}",
		Variables: []string{"customer_name", "order_number", "store_name", "reason", "tracking_url"},
		Surfaces:  []string{"orders", "finance", "messages"},
	},
	{
		Code: "WA-C07", Audience: AudienceCustomer, Category: "unavailable",
		Title: "Item tidak tersedia",
		Description: "Beritahu customer ada item yang kosong.",
		Body: "Halo {{customer_name}}, ada update untuk order {{order_number}} di {{store_name}}.\nItem berikut tidak tersedia: {{item_summary}}\nKami akan hubungi untuk opsi selanjutnya. Detail: {{tracking_url}}",
		Variables: []string{"customer_name", "order_number", "store_name", "item_summary", "tracking_url"},
		Surfaces:  []string{"orders", "purchasing", "messages"},
	},
	{
		Code: "WA-C08", Audience: AudienceCustomer, Category: "unavailable",
		Title: "Opsi item kosong (tunggu / batal / ganti)",
		Description: "Tawarkan pilihan saat item unavailable.",
		Body: "Halo {{customer_name}}, untuk order {{order_number}} di {{store_name}}, item {{item_summary}} kosong.\nPilihan: (1) tunggu restock (2) batalkan item (3) ganti produk.\nBalas pesan ini ya. Portal: {{portal_url}}",
		Variables: []string{"customer_name", "order_number", "store_name", "item_summary", "portal_url"},
		Surfaces:  []string{"orders", "messages"},
	},
	{
		Code: "WA-C09", Audience: AudienceCustomer, Category: "preorder",
		Title: "Update ETA preorder",
		Description: "Update estimasi kedatangan batch preorder.",
		Body: "Halo {{customer_name}}, update preorder {{batch_name}} di {{store_name}} untuk order {{order_number}}.\nEstimasi: {{eta_text}}\nDetail: {{tracking_url}}",
		Variables: []string{"customer_name", "batch_name", "store_name", "order_number", "eta_text", "tracking_url"},
		Surfaces:  []string{"orders", "preorders", "messages"},
	},
	{
		Code: "WA-C10", Audience: AudienceCustomer, Category: "packing",
		Title: "Sedang dikonsolidasi / packing",
		Description: "Beritahu customer order sedang dipacking.",
		Body: "Halo {{customer_name}}, order {{order_number}} di {{store_name}} sedang kami packing/konsolidasi. Kami kabari lagi saat siap kirim ya!\n{{tracking_url}}",
		Variables: []string{"customer_name", "order_number", "store_name", "tracking_url"},
		Surfaces:  []string{"packing", "orders", "messages"},
	},
	{
		Code: "WA-C11", Audience: AudienceCustomer, Category: "packing",
		Title: "Siap dikirim / diambil",
		Description: "Order packed dan siap fulfillment.",
		Body: "Halo {{customer_name}}, order {{order_number}} di {{store_name}} sudah siap ({{fulfillment_method}}).\nDetail: {{tracking_url}}",
		Variables: []string{"customer_name", "order_number", "store_name", "fulfillment_method", "tracking_url"},
		Surfaces:  []string{"packing", "orders", "messages"},
	},
	{
		Code: "WA-C12", Audience: AudienceCustomer, Category: "shipment",
		Title: "Order dikirim + tracking",
		Description: "Kirim nomor resi ke customer.",
		Body: "Halo {{customer_name}}, order {{order_number}} di {{store_name}} sudah dikirim!\nKurir: {{carrier}}\nResi: {{tracking_number}}\nLacak: {{tracking_url}}",
		Variables: []string{"customer_name", "order_number", "store_name", "carrier", "tracking_number", "tracking_url"},
		Surfaces:  []string{"shipments", "packing", "orders", "messages"},
	},
	{
		Code: "WA-C13", Audience: AudienceCustomer, Category: "shipment",
		Title: "Siap diambil (pickup)",
		Description: "Undang customer untuk pickup.",
		Body: "Halo {{customer_name}}, order {{order_number}} di {{store_name}} sudah siap diambil.\nLokasi/instruksi: {{pickup_notes}}\nKonfirmasi via chat ya. Detail: {{tracking_url}}",
		Variables: []string{"customer_name", "order_number", "store_name", "pickup_notes", "tracking_url"},
		Surfaces:  []string{"packing", "orders", "messages"},
	},
	{
		Code: "WA-C14", Audience: AudienceCustomer, Category: "shipment",
		Title: "Order selesai / diterima",
		Description: "Konfirmasi delivered atau collected.",
		Body: "Halo {{customer_name}}, order {{order_number}} di {{store_name}} sudah selesai. Terima kasih sudah belanja!\nPortal: {{portal_url}}",
		Variables: []string{"customer_name", "order_number", "store_name", "portal_url"},
		Surfaces:  []string{"orders", "shipments", "messages"},
	},
	{
		Code: "WA-C15", Audience: AudienceCustomer, Category: "address",
		Title: "Konfirmasi alamat pengiriman",
		Description: "Minta customer pastikan alamat sebelum kirim.",
		Body: "Halo {{customer_name}}, sebelum kami kirim order {{order_number}}, mohon konfirmasi alamat:\n{{shipping_address}}\nJika ada perubahan, balas chat ini atau via portal: {{portal_url}}",
		Variables: []string{"customer_name", "order_number", "shipping_address", "portal_url"},
		Surfaces:  []string{"orders", "customers", "messages"},
	},
	{
		Code: "WA-C16", Audience: AudienceCustomer, Category: "portal",
		Title: "Akses portal pelanggan",
		Description: "Bagikan link portal untuk cek order & kredit.",
		Body: "Halo {{customer_name}}! Kelola order Anda di portal {{store_name}}:\n{{portal_url}}\nLogin pakai email {{customer_email}}.",
		Variables: []string{"customer_name", "store_name", "portal_url", "customer_email"},
		Surfaces:  []string{"customers", "orders", "messages"},
	},
	{
		Code: "WA-C17", Audience: AudienceCustomer, Category: "finance",
		Title: "Refund diproses",
		Description: "Beritahu customer refund sedang/sudah diproses.",
		Body: "Halo {{customer_name}}, refund untuk order {{order_number}} di {{store_name}} sebesar {{amount_formatted}} sedang/sudah diproses.\nDetail: {{portal_url}}",
		Variables: []string{"customer_name", "order_number", "store_name", "amount_formatted", "portal_url"},
		Surfaces:  []string{"finance", "orders", "messages"},
	},
	{
		Code: "WA-C18", Audience: AudienceCustomer, Category: "finance",
		Title: "Store credit ditambahkan",
		Description: "Beritahu customer ada kredit toko baru.",
		Body: "Halo {{customer_name}}, store credit {{amount_formatted}} sudah ditambahkan di {{store_name}}.\nCek saldo: {{portal_url}}",
		Variables: []string{"customer_name", "amount_formatted", "store_name", "portal_url"},
		Surfaces:  []string{"finance", "customers", "messages"},
	},
}
