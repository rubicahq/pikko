package email

import "fmt"

func OrderConfirmationContent(storeName, orderNumber, totalFormatted, trackingURL string) (subject, html, text string) {
	subject = fmt.Sprintf("Order %s received — %s", orderNumber, storeName)
	text = fmt.Sprintf(
		"Thanks for your order at %s.\n\nOrder %s\nTotal %s\n\nTrack your order and upload payment proof:\n%s\n",
		storeName, orderNumber, totalFormatted, trackingURL,
	)
	html = fmt.Sprintf(`<!DOCTYPE html>
<html><body style="margin:0;padding:0;background:#FFFFFF;font-family:Inter,Arial,sans-serif;color:#15191E;">
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0"><tr><td align="center" style="padding:32px 16px;">
<table role="presentation" width="600" style="max-width:600px;width:100%%;border:1px solid #E1E3E8;border-radius:12px;">
<tr><td style="padding:32px;">
<p style="margin:0 0 8px;font-size:14px;color:#60646C;">%s</p>
<h1 style="margin:0 0 16px;font-size:24px;font-weight:500;letter-spacing:-0.02em;">We received your order.</h1>
<p style="margin:0 0 8px;font-size:16px;line-height:1.5;color:#60646C;">Order <strong>%s</strong></p>
<p style="margin:0 0 24px;font-size:16px;line-height:1.5;color:#60646C;">Total <strong>%s</strong></p>
<p style="margin:0 0 24px;"><a href="%s" style="display:inline-block;background:#C83200;color:#FCFCFD;text-decoration:none;padding:12px 20px;border-radius:8px;font-weight:500;font-size:14px;">Track your order</a></p>
<div style="background:#F3F3F7;border:1px solid #E1E3E8;border-radius:8px;padding:16px;">
<p style="margin:0;font-size:14px;line-height:1.5;color:#60646C;">Follow the payment instructions on your order page, then upload your transfer proof.</p>
</div>
<p style="margin:24px 0 0;font-size:12px;color:#8B8D98;">Powered by Pikko</p>
</td></tr></table>
</td></tr></table>
</body></html>`, storeName, orderNumber, totalFormatted, trackingURL)
	return subject, html, text
}

func NewOrderMerchantContent(storeName, orderNumber, customerName, totalFormatted, dashboardURL string) (subject, html, text string) {
	subject = fmt.Sprintf("New order %s — %s", orderNumber, storeName)
	text = fmt.Sprintf(
		"New order at %s.\n\nOrder %s from %s\nTotal %s\n\nOpen in dashboard:\n%s\n",
		storeName, orderNumber, customerName, totalFormatted, dashboardURL,
	)
	html = fmt.Sprintf(`<!DOCTYPE html>
<html><body style="margin:0;padding:0;background:#FFFFFF;font-family:Inter,Arial,sans-serif;color:#15191E;">
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0"><tr><td align="center" style="padding:32px 16px;">
<table role="presentation" width="600" style="max-width:600px;width:100%%;border:1px solid #E1E3E8;border-radius:12px;">
<tr><td style="padding:32px;">
<p style="margin:0 0 8px;font-size:14px;color:#60646C;">%s</p>
<h1 style="margin:0 0 16px;font-size:24px;font-weight:500;">New order received.</h1>
<p style="margin:0 0 8px;font-size:16px;color:#60646C;">Order <strong>%s</strong> from %s</p>
<p style="margin:0 0 24px;font-size:16px;color:#60646C;">Total <strong>%s</strong></p>
<p style="margin:0 0 8px;"><a href="%s" style="display:inline-block;background:#C83200;color:#FCFCFD;text-decoration:none;padding:12px 20px;border-radius:8px;font-weight:500;font-size:14px;">View order</a></p>
</td></tr></table>
</td></tr></table>
</body></html>`, storeName, orderNumber, customerName, totalFormatted, dashboardURL)
	return subject, html, text
}

func PaymentVerifiedContent(storeName, orderNumber, totalFormatted, trackingURL string) (subject, html, text string) {
	subject = fmt.Sprintf("Payment verified — %s", orderNumber)
	text = fmt.Sprintf(
		"Your payment for order %s at %s (%s) has been verified.\n\n%s\n",
		orderNumber, storeName, totalFormatted, trackingURL,
	)
	html = fmt.Sprintf(`<!DOCTYPE html>
<html><body style="margin:0;padding:0;background:#FFFFFF;font-family:Inter,Arial,sans-serif;color:#15191E;">
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0"><tr><td align="center" style="padding:32px 16px;">
<table role="presentation" width="600" style="max-width:600px;width:100%%;border:1px solid #E1E3E8;border-radius:12px;">
<tr><td style="padding:32px;">
<p style="margin:0 0 8px;font-size:14px;color:#60646C;">%s</p>
<h1 style="margin:0 0 16px;font-size:24px;font-weight:500;">Payment verified.</h1>
<p style="margin:0 0 24px;font-size:16px;color:#60646C;">Order <strong>%s</strong> · %s</p>
<p style="margin:0;"><a href="%s" style="display:inline-block;background:#C83200;color:#FCFCFD;text-decoration:none;padding:12px 20px;border-radius:8px;font-weight:500;font-size:14px;">View order</a></p>
</td></tr></table>
</td></tr></table>
</body></html>`, storeName, orderNumber, totalFormatted, trackingURL)
	return subject, html, text
}

func PaymentRejectedContent(storeName, orderNumber, reason, trackingURL string) (subject, html, text string) {
	subject = fmt.Sprintf("Payment evidence needs attention — %s", orderNumber)
	text = fmt.Sprintf(
		"Payment evidence for order %s at %s was not accepted.\nReason: %s\n\nUpload a new proof:\n%s\n",
		orderNumber, storeName, reason, trackingURL,
	)
	html = fmt.Sprintf(`<!DOCTYPE html>
<html><body style="margin:0;padding:0;background:#FFFFFF;font-family:Inter,Arial,sans-serif;color:#15191E;">
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0"><tr><td align="center" style="padding:32px 16px;">
<table role="presentation" width="600" style="max-width:600px;width:100%%;border:1px solid #E1E3E8;border-radius:12px;">
<tr><td style="padding:32px;">
<p style="margin:0 0 8px;font-size:14px;color:#60646C;">%s</p>
<h1 style="margin:0 0 16px;font-size:24px;font-weight:500;">Payment evidence rejected.</h1>
<p style="margin:0 0 8px;font-size:16px;color:#60646C;">Order <strong>%s</strong></p>
<p style="margin:0 0 24px;font-size:16px;color:#60646C;">%s</p>
<p style="margin:0;"><a href="%s" style="display:inline-block;background:#C83200;color:#FCFCFD;text-decoration:none;padding:12px 20px;border-radius:8px;font-weight:500;font-size:14px;">Upload new proof</a></p>
</td></tr></table>
</td></tr></table>
</body></html>`, storeName, orderNumber, reason, trackingURL)
	return subject, html, text
}
