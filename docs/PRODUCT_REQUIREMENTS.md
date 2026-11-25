# PIKKO - Master PRD v4.0 (historical)

> **Superseded for new work by PRD v3.0:** see `docs/PIKKO_MASTER_PRD.md`. This v4 document remains as the record of the phased build already live on `pikko.site`.

# PIKKO - Master PRD v4.0 (condensed transcription of the user's PRD; historical source for phases 0–7)

Condensed for hand-off: every requirement, list, table value, route, token and acceptance rule is kept; only prose filler is trimmed. Section numbers match the original. Copy this file to `docs/PRODUCT_REQUIREMENTS.md` in the repo.

## 0. MASTER DEVELOPMENT INSTRUCTIONS
Act as senior PM, designer, architect, FE, BE, DevOps, QA building Pikko greenfield. Production-ready commerce platform, not prototype/static demo/disconnected screens. If a repo/infrastructure exists, inspect first and reuse compatible foundations.
### 0.1 Principles
Complete end-to-end journeys; real backend; persistent DB; consistent design system; simplicity/usability; no unnecessary architectural complexity; secure tenant isolation; reusable components; business logic validated on backend; meaningful automated tests; accurate docs; never fabricate production metrics or customer info; never expose credentials; never destructive production ops without explicit approval.
### 0.2 Scope discipline
Only approved MVP. Do not add advanced features because architecture allows. Ambiguity priority: 1 customer experience, 2 merchant simplicity, 3 data integrity, 4 security, 5 maintainability, 6 dev efficiency.
### 0.3 Required docs (in docs/)
PRODUCT_REQUIREMENTS.md, DESIGN_SYSTEM.md, ARCHITECTURE.md, DATABASE_SCHEMA.md, API_REFERENCE.md, DEVELOPMENT_PROGRESS.md, REQUIREMENTS_MATRIX.md, TESTING.md, DEPLOYMENT.md. For every major task document: scope, current behavior, expected behavior, implementation steps, affected components, DB changes, relevant wireframe/screen spec, acceptance criteria, test results, known limitations. Docs must reflect actual implementation.

## 1. PRODUCT OVERVIEW
Name Pikko. Category: Simple Commerce Platform for Independent Sellers. Tagline: "Sell your way." UVP: Create a beautiful online store, give customers a better shopping experience, manage orders the way your business works. Vision: independent sellers run own commerce without technical/operational complexity. Positioning: merchant-owned commerce infrastructure. NOT: marketplace, enterprise ERP, dedicated jastip marketplace, PDF catalog converter, complex workflow automation platform, replacement for every Shopify capability. References (no proprietary assets/code): Big Cartel (simplicity), Fourthwall (branded storefront), Sellfy (onboarding), CommentSold (structured order collection), Brex (visual identity, typography, layout, discipline). Strategic principles: storefront brings merchants in; better shopping drives activation; flexible ordering differentiates; effortless operations drive retention; AI accelerates setup; complex logic feels like simple business settings.

## 2. PROBLEM
Independent sellers use Instagram, WhatsApp, TikTok, communities. Product info scattered across posts, photos, PDF catalogs, spreadsheets, chats. Customers ask price/availability/ordering/payment/status. Merchants manually answer, collect customer info, calculate totals, record orders, verify payments, consolidate purchasing, send updates. Pikko: professional storefront, structured checkout, flexible ordering, simple operational tools.

## 3. TARGET MARKET
Primary 1: Jastipers/personal shoppers (sell via social/messaging, preorder periods, buy after orders, multiple brands/locations, handling fees, purchasing schedule, ETA). Jobs: launch preorder collections, one storefront link, structured orders, verify payments, consolidate purchasing, track sourcing, manage per-customer fulfillment, reduce repetitive inquiries.
Primary 2: Social sellers (professional store, browse independently, structured orders, payments, fulfillment, purchase history).
Secondary: independent brands, handmade, creator merch, community sellers, made-to-order, small suppliers. Wholesale not MVP.
GTM: jastipers beachhead but not exclusively jastip positioning; segment landing pages, unified product.

## 4. GOALS AND METRICS
Merchant goals: create store quickly; publish without tech expertise; receive orders without re-entry; ready-stock + preorder; reduce repetitive work; manage customers/fulfillment. Customer goals: discover easily; understand price/availability/conditions; checkout without conversations; track payment/fulfillment; purchase confidently.
North Star: Completed Orders Without Manual Re-entry.
Supporting metrics: registration conversion; store creation conversion; time to store creation; time to first product; time to store publication; storefront conversion; first order activation; first successful transaction; active transacting stores; completed orders per active store; repeat purchase rate; merchant retention.
Targets (product targets, not claims): onboarding < 2 min active; first simple storefront publication < 10 min; customer checkout < 2 min; good Core Web Vitals; zero cross-store data leakage.

## 5. PILLARS
Beautiful Storefronts: themes, branding, banners, featured products, collections, browsing, product details, preview, shareable links. Flexible Ordering: ready-stock, preorder batches, deadlines, ETA, handling fees, minimum order, optional merchant approval. Effortless Operations: structured orders, payment verification, consolidated purchasing, customer records, order tracking, transactional notifications, AI-assisted catalog import.

## 6. MVP SCOPE
Modules: Account, Stores, Overview, Storefront, Catalog, Orders, Preorders, Customers, Business Rules, Settings. Surfaces: 1 marketing website, 2 passwordless auth, 3 first-store onboarding, 4 account-level store management, 5 merchant dashboard, 6 public storefront, 7 customer checkout, 8 customer order tracking, 9 transactional emails. Boundaries: marketing automation not MVP; banners/featured/collections are MVP; transactional notifications MVP; visual workflow automation post-MVP.

## 7. DOMAIN AND INFRA
Current domain pikko.site (dev, testing, controlled pilots, pre-launch). Official launch pikko.id (migrate only when ready; do NOT initiate migration without explicit approval).
| Service | Current | Future |
| Marketing | pikko.site | pikko.id |
| Dashboard | app.pikko.site | app.pikko.id |
| Storefront | shop.pikko.site | shop.pikko.id |
| API | api.pikko.site | api.pikko.id |
Validate existing DNS/infra first. Storefront URL: https://shop.pikko.site/{store-slug} (e.g. /japanpicks); slugs globally unique in storefront namespace. Merchant subdomains/custom domains post-MVP.
Dashboard routes (explicit store context): /stores, /stores/new, /stores/{store-id}/overview, /storefront, /catalog, /orders, /preorders, /customers, /business-rules, /settings, and /account/settings. Do not rely only on a global active-store cookie; multiple tabs must operate different stores independently.
Env vars (.env.example, never commit secrets): APP_ENV=development, APP_BASE_DOMAIN=pikko.site, MARKETING_URL, DASHBOARD_URL, STOREFRONT_URL, API_BASE_URL (https://pikko.site, app., shop., api.), APP_DEFAULT_LOCALE=id-ID, APP_DEFAULT_TIMEZONE=Asia/Jakarta, APP_DEFAULT_CURRENCY=IDR, DATABASE_URL, STORAGE_ENDPOINT, STORAGE_BUCKET, STORAGE_ACCESS_KEY, STORAGE_SECRET_KEY, EMAIL_PROVIDER=brevo, BREVO_API_KEY, EMAIL_FROM_NAME=Pikko, EMAIL_ACCOUNT_FROM=account@pikko.site, EMAIL_NOTIFICATION_FROM=notifications@pikko.site, EMAIL_REPLY_TO=support@pikko.site, AUTH_MAGIC_LINK_TTL_MINUTES=15, AUTH_RESEND_COOLDOWN_SECONDS=60.
Domain-agnostic: centralize URL generation for storefront, product, collection, preorder, checkout, tracking, magic links, email links, API callbacks, SEO canonicals. Store identifiers and relative paths; no hardcoded domains in business logic.
Future migration prep: DNS, TLS, app config, auth callback URLs, cookie/session behavior, CORS, payment callbacks, Brevo sender auth, SEO canonicals, sitemaps, permanent redirects, existing tracking links. Preserve store slugs and business records. Users may need to re-authenticate. Never forward sensitive auth tokens through uncontrolled redirects.

## 8. ACCOUNT AND MULTI-STORE
One account = one verified email; one account can own many stores; creating a store never creates a user. Membership model (User -> store_memberships -> Stores, each with products, orders, customers, payments). Initial role: Owner only (staff/invitations post-MVP). Account-level data: verified email, display name, language, sessions, memberships, most recently accessed store, security settings. Store-level data (independent per store): identity, storefront config, branding, theme, products, variants, inventory, collections, preorder batches, orders, payments, customers, business rules, notification settings, analytics, setup checklist. No automatic cross-store sharing. Do not hardcode one-store-per-account; configurable store limits for future subscriptions (no commercial limit approved). Every protected op validates: 1 authentication, 2 store membership, 3 membership status, 4 permissions, 5 resource ownership within store. Never trust frontend store id without authorization.

## 9. PASSWORDLESS AUTH
Email magic link only. NO password registration/login/reset, Google/social sign-in, or separate email verification. Same interface for new and returning users. Brevo delivers email; Pikko backend generates/verifies tokens and sessions.
Flow: 1 user opens auth page; 2 enters email; 3 backend validates/normalizes; 4 generates crypto-secure token; 5 stores token hash; 6 Brevo sends link; 7 user opens link; 8 user confirms sign-in via secure confirmation action; 9 backend validates and atomically consumes token; 10 creates/retrieves account; 11 establishes secure session; 12 redirect by account state. Automated GET must NOT consume token.
Screens: A Enter email - headline "Let's get started." copy "Enter your email to create an account or sign in." field Email address, CTA "Continue with email.", text "No password needed." B Check inbox - "Check your inbox." / "We've sent a secure sign-in link to your email address." show submitted email, resend countdown, resend action, use different email. C Confirm - "Continue to Pikko." CTA "Sign in securely." (prevents scanners consuming token). D Invalid/expired - "This link has expired." / "Request a new sign-in link to continue." CTA "Send a new link."
Routing: new user no stores -> first-store onboarding; saved onboarding progress -> resume; one store -> overview; multiple -> most recently accessed authorized store; previously accessed unavailable -> manage stores; no active stores -> create store.
Security: >=256 bits entropy; store hashes only; single-use; expiry 15 min; resend cooldown 60 s; rate limit by email and IP; atomic consumption; generic responses (no enumeration); secure session ids; HttpOnly cookies; Secure in prod; SameSite policy; session revocation and expiration; open-redirect protection; no raw tokens in logs; no auth URLs in analytics; no unnecessary third-party scripts on confirmation pages; disable email click tracking for magic links where supported.
Account creation only after successful verification; prevent duplicate accounts for same normalized email; handle concurrent requests; a requested link is not a registration.
Sessions: sign in/out, current session identification, active session management, revocation, sign out all devices. Lifetime 30 days (configurable, subject to security review). Sensitive account changes may require fresh magic link.
Acceptance: email-only auth; same UI new/returning; Brevo delivery; expired links unusable; consumed links unusable; scanners don't consume; unauthorized sessions rejected; no password UI; account creation idempotent.

## 10. FIRST-STORE ONBOARDING
Objective: first store with minimal friction, < 2 min active interaction; do not force full configuration before dashboard.
Step 1 Store identity ("Create your first store."): store name (2-80 chars, Unicode ok), store slug (suggested from name, editable; lowercase letters, numbers, hyphens; 3-50 chars; no leading/trailing hyphen; globally unique; reserved names prohibited: admin, api, app, support, login, checkout, settings, www, ... centralized reserved-slug list; validate availability on backend), show `shop.pikko.site/{store-slug}`, business category (Personal shopper / Jastip; Fashion and accessories; Beauty and skincare; Food and beverages; Home and living; Baby and kids; Electronics and gadgets; Handmade and crafts; Collectibles and hobbies; Health and wellness; Merchandise; Other) used for personalization/analytics only. CTA "Continue."
Step 2 Selling model ("How do you sell?"): Ready Stock (available and ready to ship), Preorder (collect orders before sourcing/preparing), Both. Selectable cards with simple outline icons; selected = soft orange background + orange border + clear indicator. Default preference, changeable later. CTA "Create my store."
Store creation: store + owner membership in ONE DB transaction. Defaults: theme Pikko Signature, currency IDR, timezone Asia/Jakarta, locale Bahasa Indonesia, publication Draft, checkout Disabled until required config complete. Redirect to store overview.
Excluded from mandatory onboarding: logo, description, address, WhatsApp, bank account, product upload/images, theme selection, subscription, payment gateway. Collect progressively.
Progress: persist completed steps; back; resume; correct before creating; prevent duplicate stores on repeated submission.
Visual: desktop minimal split-screen (form + relevant storefront preview), white/light neutral, large Inter headings, restrained orange. Mobile single column, visible progress indicator, accessible inputs, no decorative panels.
Acceptance: first store in two short steps; slug availability validated; interrupted onboarding resumable; duplicate submissions no duplicate stores; ownership established; merchant reaches dashboard immediately.

## 11. MULTI-STORE MANAGEMENT
Store switcher persistent near top of sidebar: current store logo/initial, name, dropdown indicator, authorized stores, current selection, Create new store, Manage stores, search when many. Switching: validate authorization; update route; load config; refresh store data; isolate previous store caches; persist last accessed store; warn on unsaved changes; never change another tab's store context.
Additional stores from switcher and Manage stores page; reuse the two-step flow; no re-verification; each new store has independent config, default theme, independent checklist, empty catalog/customers/orders; no copying from other stores.
Manage stores page (list or grid): logo, name, storefront URL, selling model, publication status, setup progress, created date. Actions: open dashboard, preview storefront, manage settings, create new store, archive store.
Archiving: explicit confirmation; disable new orders; remove storefront from public browsing; preserve historical orders, payments, customers, tracking access; prevent archiving when unresolved transactions need merchant action or provide controlled resolution; permanent deletion not an ordinary action.
Acceptance: multiple stores per user; independent records; switching loads correct context; unauthorized stores inaccessible; multi-tab safe; creating store doesn't alter others.

## 12. MERCHANT DASHBOARD
Nav: Overview, Storefront, Catalog, Orders, Preorders, Customers, Business Rules, Settings. Account settings and store management visually distinct from store settings.
Overview: store name, publication status, storefront link, date range, operational tasks, recent orders, active preorder batches, setup checklist. Key metrics: total order value, total orders, verified payment value, unique purchasing customers; distinguish unpaid order value from received payment value; never label unpaid as revenue. Operational tasks (only from real data): orders requiring confirmation, awaiting payment, payment evidence to verify, ready for fulfillment, preorder batches near deadline, purchasing items needing attention. Quick actions: add product, create preorder, customize storefront, view orders, share storefront.
Setup checklist derived from persisted state (not manual checkboxes), dismissible/reopenable, optional tasks don't block publication. Ready-stock: create store, first product, customize storefront, payment methods, configure fulfillment, publish and share. Preorder: create store, first product, first preorder batch, customize storefront, payment methods, fulfillment, publish and share. Both: relevant union.
Publication vs checkout readiness are separate: may publish for browsing before payments; live checkout disabled until payment and fulfillment valid; clear customer-facing message when checkout unavailable.

## 13. DESIGN SYSTEM
Primary visual reference https://www.brex.com/ (character, not functionality; no proprietary logos/assets/copy/code). Applies to marketing, auth, onboarding, dashboard, default storefront, checkout, tracking, transactional emails. Tokens below are Pikko's approved Brex-inspired values.
Personality: premium, modern, minimal, approachable, confident, professional, energetic, precise; avoid generic AI-generated SaaS look. Principles: strong typography over decoration; generous whitespace; precise alignment; white/cool neutral surfaces; restrained orange; subtle borders; minimal shadows; high-quality product visuals; clear action hierarchy; consistent components.
Palette: brand primary #FF3D00; hover #EB3800; pressed #D63200; accessible dark #C83200; soft #FFF1EB; border #FFD2C2; background #FFFFFF; subtle #F9F9FB; surface #F3F3F7; surface strong #E8E9EE; text primary #15191E; secondary #60646C; muted #8B8D98; subtle #B9BBC6; inverse #FCFCFD; dark #000710; border #E1E3E8; border strong #C7CAD2.
Distribution guideline: white/light 75%, charcoal/dark 15%, secondary neutral 5%, orange 5%. No solid-orange sidebars, excessive orange cards, orange on every icon.
Semantic: success fg #166534 bg #F0FDF4 border #BBF7D0; warning #92400E / #FFFBEB / #FDE68A; error #B91C1C / #FEF2F2 / #FECACA; info #1D4ED8 / #EFF6FF / #BFDBFE; neutral #4B5563 / #F3F4F6 / #E5E7EB. Don't confuse brand orange with warning/error.
Accessibility WCAG 2.2 AA: 4.5:1 normal text, 3:1 large text/interactive boundaries, visible focus, keyboard nav, labels, accessible dialogs, alt text, no color-only status, reduced motion. Signature orange fails with white small text: use dark text on #FF3D00 or white on accessible darker orange (#C83200). Validate foreground/background combos programmatically.
CSS tokens (centralized): --brand-primary #FF3D00; --brand-hover #EB3800; --brand-pressed #D63200; --brand-accessible-dark #C83200; --brand-soft #FFF1EB; --brand-border #FFD2C2; --background #FFFFFF; --background-subtle #F9F9FB; --surface #F3F3F7; --surface-strong #E8E9EE; --foreground #15191E; --foreground-secondary #60646C; --foreground-muted #8B8D98; --foreground-inverse #FCFCFD; --dark #000710; --border #E1E3E8; --border-strong #C7CAD2; --success #166534; --warning #92400E; --error #B91C1C; --info #1D4ED8; --radius-xs 4px, sm 6, md 8, lg 10, xl 12, 2xl 16; --font-sans "Inter", Arial, sans-serif. Map Tailwind semantic utilities to tokens; no arbitrary hardcoded colors in components.
Typography: Inter variable (next/font or self-hosted), fallback Arial; single typeface everywhere; no Brex proprietary fonts. Marketing (desktop/mobile px, weight, line-height): Display XL 72/42 500 1.00; Display L 56/36 500 1.05; Display M 48/30 500 1.05; H1 36/28 500 1.12; H2 30/24 500 1.20; H3 24/20 500 1.25; Body large 18/16 400 1.55; Body 16/14 400 1.50; Supporting 14/14 400 1.50; Label 12/12 500 1.40. Dashboard: Page title 30/500/36; Section title 20/500/28; Card title 16/500/24; Body 14/400/21; Button 14/500/20; Form label 13/500/20; Table cell 14/400/20; Table heading 12/500/18; Metadata 12/400/18; KPI value 32/500/38. Rules: body 400, headings 500, selective emphasis 600, avoid excessively bold; marketing headlines -0.02em tracking; tabular numerals for money/quantities; sentence case; avoid long uppercase headings and tiny text.
Spacing: 4px base; approved values 4, 8, 12, 16, 20, 24, 32, 40, 48, 64, 80, 96, 128. Label-input 8; field gap 20-24; card padding 20-24; table cell 12-16; dashboard section gap 24; desktop page padding 32; mobile 16; marketing section 96 desktop, 48-64 mobile.
Radius: badge 4-6; button 8; input 8; dropdown 8; table container 10; card 10; marketing panel 12-16; modal 12; product image 8-10; avatar full. Avoid pill-shaped components and large radii everywhere.
Shadows: xs 0 1px 2px rgba(0,7,16,.04); sm 0 2px 6px rgba(0,7,16,.05), 0 1px 2px rgba(0,7,16,.03); md 0 8px 24px rgba(0,7,16,.08), 0 2px 6px rgba(0,7,16,.04); overlay 0 16px 48px rgba(0,7,16,.12). Borders define boundaries; avoid heavy shadows, floating cards, neon glows, glassmorphism, decorative gradients.
Icons: Lucide, stroke 1.75; 16 small, 20 standard, 24 large, 32+ feature; charcoal/secondary gray, orange selectively; no mixed styles.
Layout: Marketing max 1280, standard 1200, reading 680-760, 12/8/4 columns, gutters 24-32 desktop 16 mobile. Dashboard sidebar 248 (collapsed 64), top bar 64, padding 32/24/16, content max 1440. Auth form max 420, centered or split. Checkout max 1120, two-column desktop, single-column mobile.
Breakpoints: mobile <640; 640; 768; 1024; 1280; 1536. Validate at 375, 390, 768, 1024, 1440; no horizontal overflow.
Motion: hover 120-160ms; standard 180-220; dialog 200-240; marketing reveal <=400; subtle opacity/position; respect reduced motion; no continuous decorative animation.
Imagery: editorial, realistic; original merchant photography, real product screenshots (only implemented features), natural light. Avoid AI-generated people, abstract 3D, excessive gradients, unrealistic dashboards, fake functionality, unrelated stock.
Brand: original Pikko wordmark; temporary clean typographic wordmark until approved logo; do not imitate Brex logo.

## 14. COMPONENT LIBRARY
Tailwind, Radix UI, shadcn/ui where appropriate, Lucide; customize everything, no default styling shipped.
Buttons: Primary, Secondary, Outline, Ghost, Destructive, Text link. Primary: #FF3D00 with accessible dark text, radius 8, weight 500, height 40 (large 48). Secondary: #000710 bg, white text. Outline: white bg, neutral border, charcoal. States: hover, pressed, focus-visible, disabled, loading; prevent duplicate submission.
Inputs: height 44 (compact 36), radius 8, white, neutral border, 14px, padding 12; visible labels; field-specific validation messages; clear focus/error states.
Cards: white, 1px neutral border, radius 10, padding 20-24, minimal/no shadow; don't wrap everything in cards.
Tables: white, subtle neutral header, ~56px rows, horizontal dividers, subtle hover, left-aligned text, right-aligned numbers, pagination, search, filters, sort indicators, no heavy vertical lines.
Status badges (compact semantic with text): Paid, Awaiting payment, Confirmed, Draft, Open, Closed, Cancelled, Failed.
Navigation: desktop sidebar white/light neutral with thin right border, wordmark, store switcher, outline icons, subtle selected background, account controls at bottom. Mobile: compact header, accessible drawer.
Dialogs widths 400/480/640/max 800; drawers 480-640 desktop, full-screen mobile; focus trap, keyboard dismissal, accessible titles.
Tabs/filters: compact, underline or subtle neutral; filters in URL query params; reset action.
Empty states explain what is missing, why it matters, action (e.g. "You haven't received any orders yet." / "Share your storefront to start receiving orders." CTA Share storefront).
Loading: skeletons matching layout; no full-page spinners for routine loads; progress for long imports.
Feedback: concise toasts; persistent inline errors when action needed; never show success before backend confirms.
Charts: primary #FF3D00, secondary #15191E, supporting #8B8D98, grid #E8E9EE; simple; never fabricate metrics.
Component preview page (dev only) documenting variants, sizes, states, a11y, responsive, usage; must not be exposed in production.

## 15. MARKETING WEBSITE
Convert independent sellers into registered users; Brex-like polished editorial language. Nav: Product, Solutions, Pricing, Resources, Sign in, Get started (hide unavailable pages, no dead links).
Homepage: Hero - eyebrow "Commerce for independent sellers."; headline "Your business. Your store. Your way."; copy "Create a beautiful online store, make shopping easier for your customers, and manage orders the way your business works."; CTAs "Create your store." / "Explore Pikko."; large medium-weight Inter headline, white/subtle neutral bg, orange primary CTA, whitespace, realistic storefront/dashboard visuals. Pillars: Beautiful storefronts, Flexible ordering, Effortless operations (editorial sections, large product visual panels). Beautiful storefronts: theme customization, collections, featured products, mobile shopping, branding. Flexible ordering: ready stock, preorder batches, deadlines, ETA, handling fees. Effortless operations: order management, payment verification, consolidated purchasing, customer tracking. Solutions: jastipers/personal shoppers, social sellers, independent brands. How it works: 1 Create your store, 2 Add your products, 3 Share your store, 4 Start receiving orders. Pricing: clean layout, do NOT publish unapproved pricing/plans/terms/trials. Final CTA dark section: "A better way to sell starts here." CTA "Create your store." Footer: Product, Solutions, Support, Contact, Privacy policy, Terms.
Solution pages (Jastipers, Social sellers, Independent brands): segment problems, capabilities, example journey, operational benefits, real screenshots, registration CTA.
Integrity: do not invent merchant counts, revenue, testimonials, customer logos, performance claims, features, commercial terms.

## 16. STOREFRONT MANAGEMENT
Branding: logo, name, description, contact email, WhatsApp, social links, favicon, theme, primary/background/text colors.
24 curated themes (Background / Text / Primary): Pikko Signature #FFFFFF/#15191E/#FF3D00; Mono Studio #FFFFFF/#171717/#171717; Warm Handmade #FFF8F0/#3D3028/#A65E36; Nordic Living #F7F6F2/#292B28/#52634D; Soft Baby #FFF7FA/#503A45/#A84E76; Modern Baby #F3F8FC/#263D52/#426B92; Bold Streetwear #171717/#FAFAFA/#E7C94E; Classic Menswear #F4F4F2/#263039/#354C60; Clean Beauty #FFF9F6/#473530/#B85046; Botanical #F4F7EF/#233E2C/#426A45; Earth Studio #F7F1E8/#45392E/#936347; Ocean Breeze #F1F8FA/#193E4D/#267489; Coastal #F8F6F0/#30434A/#326F86; Premium Noir #121212/#F7F5EF/#C6A467; Quiet Luxury #FAF8F3/#342D29/#786247; Cherry Pop #FFF5F5/#4C202A/#B62840; Tangerine #FFF7ED/#402C20/#D64B16; Lavender #F8F5FF/#382D54/#7853B0; Fresh Market #F8FBEF/#344020/#56762B; Retro Club #FBF1DC/#573C31/#B44D32; Chocolate #F8EEE5/#3E271C/#865336; Midnight Blue #101C2B/#F5F8FC/#87B9EE; Sakura #FFF3F6/#583642/#AD5573; Contemporary #F4F4F7/#272A35/#5057A6.
Derive supporting surfaces/borders via validated theme definitions; auto-determine accessible button foreground; validate all combinations; merchants can customize the three colors; merchant themes never affect the Pikko dashboard.
Default storefront structure: 1 header, 2 banner, 3 active preorder collections, 4 featured products, 5 featured collections, 6 product browsing, 7 store information, 8 footer; hide sections without published content.
Banners: up to 3 active; fields desktop image, optional mobile image, headline, supporting text, optional CTA, internal destination (product/collection/preorder batch), display order, active status; ratios desktop 16:6, mobile 4:5 or 1:1; crop preview; no auto-advance by default.
Featured products up to 8 (select, order, remove, preview; only published products public). Featured collections up to 4 (select, order).
Editor desktop: left panel (branding, theme, banners, featured products, featured collections, store information), right panel live preview (desktop/mobile, publication status); dedicated preview mode on mobile.
Draft vs published configuration separated; editing draft doesn't affect live; publish applies validated draft atomically; discard unpublished changes.
Preview: secure preview for unpublished stores; URLs must not expose unpublished products to unauthorized users; noindex.
Acceptance: no-code customization; all 24 themes; accessible contrast; banners/featured work; drafts private; published changes consistent; mobile usable.

## 17. CATALOG
Products: create, edit, duplicate, archive, publish, unpublish, search, filter, bulk publication updates. Fields: name, description, images, SKU, brand, category, tags, base price, optional compare-at price, optional internal cost (never public), ready-stock eligibility, preorder eligibility, availability, stock quantity, optional weight, publication status.
Variants: merchant-defined option names (Size, Color, Material); variant SKU, option combination, price override, stock, availability, optional image; prevent duplicate combinations.
Inventory: finite or unlimited, variant-level stock, reservations, stock adjustment history; no negative stock; server-side validation.
Images: multiple, primary, ordering, removal, alt text; configurable upload limits; optimized for storefront; preserve originals where needed.
Collections: products in multiple collections; name, description, cover image, assigned products, display order, publication status. Collections = merchandising; preorder batches = operational; don't merge logic.
Product list: thumbnail, name, SKU, price, stock, selling model, publication status, last updated; pagination, filtering. Editor sections: basic info, media, pricing, variants, inventory, selling model, collections, publication; two-column desktop.
Integrity: historical orders preserve original product info; archiving/editing never changes historical orders.

## 18. AI CATALOG IMPORT
Accelerator, not primary positioning. Entry: Add manually; Import PDF; Import CSV (when available).
PDF flow: upload; validate type/size; async import job; AI extracts items per page; page images kept for flipbook (no product-image crop path); review interface; merchant edits/removes; confirms selected; saved as DRAFT products; merchant publishes later. Fields: name, description, brand, price, SKU, category, variant info. Do not fabricate missing values; flag uncertain results.
Limits (configurable defaults): one PDF per job; 25 MB; 50 pages; 500 items; validate vs infra/AI provider; not commercial entitlements.
Status: Pending, Processing, Review required, Completed, Failed; progress and retry.
Review UI: image, name, price, brand, SKU, warnings, editable fields, checkbox; bulk category/tag assignment.
AI security: uploaded content untrusted; don't execute embedded instructions; no access to credentials/tools/other stores; structured extraction schemas; validate AI output before persistence.
CSV: template download, validation, column mapping, preview, duplicate detection, error report, draft creation; implement after manual product management and PDF import are stable.

## 19. CUSTOMER STOREFRONT
Feels like professionally branded independent store; not spreadsheet/dashboard/static PDF/unstructured list.
Header: logo or name, product navigation, collections, search, cart, optional contact; compact on mobile. Browsing: grid, collections, search, category filter, availability filter, preorder browsing; grid 4 columns large desktop, 3 standard desktop, 2 mobile. Product cards: image, name, price, optional compare-at, availability, preorder indicator; imagery focus; few badges, light borders.
Product detail: gallery, name, price, description, variant selector, quantity, availability, selling model, preorder info, add to cart; desktop images left/info right; mobile images above, accessible purchase action. Preorder info: batch name, order deadline, ETA, payment policy, merchant notes; if product in multiple batches require batch selection. Contact: merchant contact links, WhatsApp optional; never force WhatsApp for standard orders.

## 20. CART AND CHECKOUT
Cart: multiple products, variant details, quantity updates, removal, subtotal, handling fees, shipping fees, final total; persisted without registration; cart doesn't guarantee inventory.
Fulfillment groups: separate incompatible models (ready-stock, preorder batch A, batch B); multiple cart groups, each checked out independently in MVP; explain grouping; never combine incompatible schedules.
Guest checkout: name, email, phone/WhatsApp, fulfillment method, shipping address where required, optional notes, payment method, acceptance of purchase terms.
Layout: desktop customer/fulfillment left + summary right; mobile single column, accessible summary, clear final action; no marketing distractions.
Server-side validation: store publication, checkout readiness, product publication, variant availability, stock, preorder quota, batch open/close timestamps, selling model compatibility, product price, handling fees, minimum order, shipping policy, payment policy, approval requirements. Never trust client totals.
Price changes: recalculate, inform customer, require confirmation of updated amount; never silently submit a higher total.
Order creation transactional: customer info, items, price snapshots, fees, fulfillment info, applicable rule versions, inventory reservations, initial statuses; idempotency keys prevent duplicates.
Confirmation: order reference, summary, payment instructions, payment status, order status, expected fulfillment, secure tracking link; send Brevo transactional email.

## 21. FLEXIBLE ORDERING
Ready stock: Browse > Cart > Checkout > Payment > Fulfillment > Completed; stock validation, reservation, optional approval, payment verification, shipping, pickup.
Preorder: Browse preorder > Select > Checkout > Confirmation if required > Payment > Purchasing > Arrival > Fulfillment > Completed; first-class capability; multiple batches alongside ready-stock.
Fulfillment methods MVP: flat-rate shipping, free shipping, customer pickup. Config: enabled methods, shipping fee, pickup instructions, service area/restrictions. Never silently assign zero shipping when fee unknown; complete fulfillment pricing required before live checkout. Carrier integrations post-MVP.
Reservation: for finite stock and preorder quotas reserve transactionally on order acceptance; configurable payment deadlines; release on eligible cancellation/expiry; no duplicate reservations; reservation history; safe under concurrency. Orders awaiting approval may reserve per configurable confirmation deadline. Do not auto-expire orders with payment evidence awaiting verification. Merchant sees reserved quantities.
Post-MVP selling models: custom orders, sourcing requests, quotations, made-to-order, deposits (keep extensible, don't build).

## 22. PREORDER BATCHES
Batch = defined selling and purchasing period (e.g. Japan December Shopping Batch: opens 1 Dec, closes 10 Dec, shopping 12 Dec, ETA 22 Dec).
Fields: name, slug, description, cover image, opening timestamp, closing timestamp, optional purchasing date, ETA date/range, merchant notes, assigned products, eligible variants, quantity limits, price overrides, payment policy, publication status, operational status.
Lifecycle: Draft, Scheduled, Open, Closed, Purchasing, Awaiting arrival, Arrived, Fulfilling, Completed, Cancelled; validated transitions; batch status separate from order status.
Product assignment: product can be in many batches; per-assignment batch price, eligible variants, quantity limit, availability, notes; no duplicate product masters; historical prices unchanged.
Auto open/close: persistent scheduled jobs or periodic checks; backend checkout independently enforces timestamps. On close: stop new orders, preserve existing, update storefront, notify merchant, update views. Merchants may close early; reopening requires explicit action and valid revised closing timestamp.
Consolidated purchasing list aggregated by product, variant, SKU, brand, required, purchased, outstanding, unavailable. Default requirement = confirmed and fully paid orders; exclude cancelled, rejected, unpaid (show unpaid demand separately).
Actions: record purchased quantities, record unavailable quantities, view outstanding, identify affected orders, allocate purchased quantities to order items, export CSV; no allocations exceeding purchased or ordered quantities.
Unavailable items: record quantity, identify affected customers, record merchant resolution, notify customers, record cancellation/refund info, preserve history; never silently modify paid totals.
Batch progress updates send configured notifications; batch arrival does not auto-mark individual orders delivered.
Batch list: cover, name, deadline, order count, ordered item quantity, payment summary, operational status. Detail tabs: Overview, Products, Orders, Purchasing, Activity.
Acceptance: simultaneous batches; closed batches reject orders; batch prices applied; quota enforced; aggregation accurate; cancelled excluded; unavailable traceable; batch and order fulfillment independent.

## 23. ORDER MANAGEMENT
List: search, date range, order/payment/fulfillment status, selling model, batch, customer filter, pagination; shows reference, customer, date, amount, three statuses, batch.
Detail: customer info, items, variants, quantities, price snapshots, fees, payment records, fulfillment details, shipping address, customer notes, internal notes, activity history.
Three separate status dimensions. Order: Pending confirmation, Confirmed, Completed, Cancelled. Payment: Unpaid, Awaiting verification, Paid, Failed, Refunded, Partially refunded. Fulfillment: Unfulfilled, Purchasing, Awaiting arrival, Ready to ship, Ready for pickup, Shipped, Delivered, Collected, Cancelled.
Transition rules: rejected -> Cancelled with reason; payment verification needs eligible order; fulfillment needs appropriate confirmation and payment; completed needs fulfilled and settled; cancelled can't receive ordinary fulfillment updates; paid cancellation requires refund resolution; backend rejects invalid transitions.
Merchant actions: confirm, reject, verify payment, reject evidence, update fulfillment, add tracking, record pickup, cancel, record refund, internal notes; confirm sensitive actions.
Immutable snapshots: product name, SKU, variant, unit price, quantity, fees, shipping, batch info, applied rule versions, customer info, fulfillment info; separate adjustment records for post-order financial changes.
Audit: actor, action, previous state, new state, timestamp, reason; not exposed publicly.

## 24. PAYMENTS
MVP: manual bank transfer (must work without gateway) + optional approved gateway (provider TBD).
Bank transfer config per store: bank name, account holder, account number, instructions, payment deadline; enable/disable accounts; never share across stores.
Flow: receive reference; view instructions; transfer; submit evidence; merchant reviews; verifies or rejects; customer notified.
Evidence: secure image/PDF upload; private storage; authorized access; expiring signed URLs; file validation; size limits; safe preview; audit history; not settlement until verified.
Policies MVP: full payment after checkout; full payment after merchant confirmation. Post-MVP deposits, partial, installments. Payment records must support multiple attempts and future schedules.
Verification: view evidence, verify, reject with reason; prevent duplicate verification; verified payment updates payment status transactionally.
Refunds (manual): amount, reason, external reference, merchant confirms completion; don't imply Pikko transferred funds. Future gateway refunds use provider-confirmed results.
Gateway foundation: provider abstraction; server-side payment creation; signature verification; idempotent callbacks; reconciliation; expiry; failure handling; refund status; never store raw card data; don't enable unconfigured/unapproved provider.

## 25. CUSTOMERS
Auto-create store-scoped customer from valid orders: name, email, phone, shipping addresses, first/last order date, total orders, completed order value, internal notes. Conservative matching with normalized email/phone; never merge by name only; conflicting identifiers don't auto-merge; never share across stores. List: name, email, phone, total orders, completed value, last order; search, pagination. Detail: contact, order history, payment history, completed value, notes. Customer portal post-MVP; guest tracking MVP; customers never need merchant accounts.

## 26. ORDER TRACKING
Secure unpredictable token per order (hash stored where appropriate), sent by email, no registration. Page: merchant identity, reference, summary, order/payment/fulfillment status, batch progress, ETA, payment instructions, evidence submission, shipment tracking, merchant contact. Security: high-entropy tokens, rate limit, noindex, prevent URL leakage, invalidation support, minimal personal info. Never expose internal notes, product costs, other customers, purchasing lists, private operations. Visual: clean mobile-first timeline with meaningful steps; merchant theme colors where appropriate.

## 27. BUSINESS RULES
No visual workflow canvas in MVP. Nav: handling fees, minimum order, order approval, rule preview, notification preferences.
Handling fees: fixed per order or percentage of eligible product subtotal; scopes: entire store, ready-stock, preorder, specific batch; displayed before checkout. Minimum order: min product subtotal, min item quantity; scopes store/selling model/batch; validated against the cart group. Approval: require confirmation before payment; same scopes.
Precedence per rule type: 1 batch, 2 selling model, 3 store default; more specific overrides; one effective handling fee per checkout; prevent ambiguous duplicate rules in same scope.
Calculation order: 1 resolve product/variant prices; 2 batch price overrides; 3 product subtotal; 4 validate minimum order; 5 handling fee; 6 shipping fee; 7 final total. Percentage fee applies to eligible product subtotal excluding shipping/other fees. Deterministic rounding.
Rule preview simulator: inputs selling model, batch, sample subtotal, sample quantity, fulfillment method; outputs applicable rules, handling fee, minimum result, approval requirement, shipping fee, final total; MUST use same backend evaluation as live checkout.
Versioning: published rules versioned; applied version and calculated results stored on each order; changing a rule doesn't alter history. Future workflow engine extensibility (triggers, conditions, actions, scheduled actions, branching, integrations); no arbitrary merchant code.

## 28. BREVO EMAIL
Brevo Transactional Email API. Senders (verified only, don't assume configured): account@pikko.site (magic links/security), notifications@pikko.site (orders/payments/fulfillment), support@pikko.site, hello@pikko.site (future).
Magic link email: subject "Your Pikko sign-in link"; "Welcome to Pikko." "Use the button below to securely access your account." CTA "Continue to Pikko." "This link expires in 15 minutes and can only be used once." "If you did not request this email, you can safely ignore it." No promotional content.
Merchant notifications: new order, order needing confirmation, payment evidence submitted, batch closed, purchasing updates, fulfillment reminders. Customer notifications: order received, confirmed, rejected, payment instructions, payment verified, evidence rejected, preorder progress, product unavailable, shipped, ready for pickup, cancelled, refund recorded/confirmed.
Email design: max 600px, white, Inter with Arial fallback, text #15191E / #60646C, orange CTA with accessible foreground, info panels #F3F3F7, restrained borders, minimal radius, responsive, plain-text alternative; customer emails show merchant store identity prominently, Pikko as supporting platform.
Localization: Bahasa Indonesia (default) and English.
Service architecture: centralized email service with Brevo adapter; persistent delivery jobs; async; templates; variables; attempt history; provider message ids; retry policies; idempotency; webhook processing; bounce and complaint handling; failure logging.
Auth email reliability: high priority; expiry accounts for delivery timing; never send an expired link; clear resend UX. Transactional reliability: email failure never rolls back a commerce transaction; persistent dedup keys; handle ambiguous provider responses conservatively; no blind retries that may duplicate.
Domain auth: verified senders, DKIM, DMARC, SPF, Brevo records; authenticate pikko.id before launch; sender domains independently configurable. MVP transactional only; marketing post-MVP.

## 29. IN-APP NOTIFICATIONS
Store-scoped notifications: new orders, pending confirmations, payment evidence, closed batches, purchasing issues. Center: unread count, list, mark read, mark all read, link to resource; respects selected store. Preferences for optional operational notifications; essential security emails cannot be disabled.

## 30. SETTINGS
Account: verified email, display name, language, active sessions, sign out, sign out all, store memberships; no password settings. Email change: verify new address securely; require recent authentication or confirmation of existing identity; prevent duplicates; notify previous email.
Store settings: identity, slug, category, selling model, contact info, locale, timezone, currency, payment config, fulfillment config, notification preferences, publication, archiving. Slug changes: validated; keep previous slugs as redirect aliases where feasible; prevent others immediately claiming historical slugs; show implications.

## 31. TECHNICAL ARCHITECTURE
Frontend: Next.js, TypeScript, Tailwind, accessible primitives, Inter variable, Lucide. Backend: Go, versioned REST, modular services. DB: PostgreSQL. Storage: S3-compatible or existing. Email: Brevo. Jobs: persistent PostgreSQL-backed queue or equivalent. Deployment: VPS, reverse proxy, HTTPS, env config. Redis optional; no unnecessary infrastructure.
Modular monolith domains: Authentication, Accounts, Stores, Storefront, Catalog, Inventory, Checkout, Orders, Preorders, Payments, Customers, Business Rules, Notifications, Analytics, File imports. No microservices.
Suggested structure: pikko/ apps/web, services/api, packages/{ui,design-tokens,shared-types}, database/{migrations,seeds}, infrastructure, docs, tests (adapt if simpler).
Frontend requirements: reusable components, typed API clients, centralized tokens, predictable fetching, store-scoped cache keys, form validation, loading/error states, responsive, accessible, SEO-friendly public rendering.
Backend requirements: auth middleware, store authorization middleware, request validation, domain services, repository layer, transactions, idempotency, structured logging, consistent errors, background jobs.
Money: integer minor units or fixed-precision decimal; never floats; IDR whole rupiah; persist currency codes; deterministic rounding. Time: UTC storage, display in store timezone (default Asia/Jakarta); batch open/close by absolute timestamps.
Multi-tenancy: store is the tenant; every store-owned resource carries/inherits store id; store-scoped queries, uniqueness, cache keys, storage paths; membership authorization; cross-store access tests.

## 32. DATABASE (logical; do not create unnecessary tables)
Account/auth: users, authentication_tokens, sessions, user_preferences, stores, store_memberships, store_slug_aliases, onboarding_drafts. Storefront: storefront_settings, storefront_versions, storefront_banners, storefront_featured_products, storefront_featured_collections. Catalog: products, product_images, product_options, product_option_values, product_variants, variant_option_values, collections, collection_products, inventory_adjustments, inventory_reservations. Preorders: preorder_batches, preorder_batch_products, preorder_batch_variants, preorder_purchasing_records, preorder_item_allocations. Customers: customers, customer_addresses. Checkout/orders: checkout_sessions, orders, order_items, order_adjustments, order_status_history, order_tracking_tokens, shipments. Payments: store_payment_methods, payments, payment_evidence, refunds. Rules: business_rules, business_rule_versions, order_rule_applications. Imports: catalog_import_jobs, catalog_import_items, stored_files. Ops: notifications, notification_preferences, notification_deliveries, email_templates, background_jobs, event_outbox, audit_logs, analytics_events.
Integrity: PKs, FKs, unique constraints, store-scoped indexes, transactional updates, historical snapshots, valid state constraints, safe archival. Migrations: version-controlled, backward-compatible where practical, tested outside production, documented recovery, no destructive production migration without approval.

## 33. API
Auth: POST /api/v1/auth/magic-link/request; POST /api/v1/auth/magic-link/verify; GET /api/v1/auth/session; POST /api/v1/auth/logout; POST /api/v1/auth/logout-all.
Account/stores: GET,PATCH /api/v1/account; GET,POST /api/v1/stores; GET,PATCH /api/v1/stores/{store-id}; POST /api/v1/stores/{store-id}/archive.
Store-scoped under /api/v1/stores/{store-id}/: storefront, products, collections, inventory, preorders, orders, payments, customers, business-rules, catalog-imports, notifications, analytics.
Public: GET /api/v1/public/stores/{slug}; /products; /products/{product-slug}; /collections; /preorders; POST /checkout/preview; POST /checkout/submit. Public endpoints expose only published/permitted data.
Tracking: token-authorized endpoints for tracking, payment instructions, evidence submission, customer actions; never sequential-id authorization.
Webhooks: POST /api/v1/webhooks/brevo (validate authenticity); future provider-specific payment webhooks.
Standards: consistent status codes, structured errors, request ids, pagination, filtering, validation, authorization, idempotency, OpenAPI docs.

## 34. EVENTS
Transactional outbox (or equivalent), no broker in MVP. Envelope: event_id (uuid), event_name, event_version, store_id, occurred_at (ISO-8601), source, actor{}, subject{}, correlation_id, idempotency_key, properties{}; account-level events may omit store_id.
Events: account: authentication_completed, account_created, store_created, additional_store_created, store_archived. Storefront: storefront_published, storefront_unpublished. Catalog: product_created, product_published, product_archived, catalog_import_requested/completed/failed. Preorder: preorder_batch_created/opened/closed/status_updated, preorder_item_unavailable. Order: order_created/confirmed/rejected/cancelled/completed. Payment: payment_evidence_submitted, payment_verified, payment_rejected, payment_received, payment_refunded. Fulfillment: fulfillment_status_updated, order_shipped, order_delivered, order_collected. Customer: customer_created, customer_updated.
Reliability: persist events with business transactions; idempotent processing; retry transient failures; record permanent failures; no duplicate side effects; preserve correlation ids; debuggable. Pricing, availability, authorization, checkout validation stay synchronous.

## 35. SECURITY AND PRIVACY
Secure auth/sessions, store authorization, input validation, output encoding, CSRF where applicable, rate limiting, safe uploads, private object storage, signed URLs, HTTPS, security headers, secret management, audit logging. Cross-store tests for products, orders, customers, payments, payment evidence, preorders, imports, analytics, store settings, notifications; both direct access and list endpoints. Never expose DB credentials, Brevo keys, auth tokens, other stores' data, internal costs, internal notes, private payment evidence, unnecessary customer info. Files: validate signatures/types, restrict sizes, block executables, safe storage paths, separate public assets from private documents, scan where supported, prevent unauthorized access. Privacy: Indonesian privacy requirements; privacy notices, data access/correction/deletion procedures, retention, transaction record preservation; technical implementation is not legal approval.

## 36. PERFORMANCE AND SEO
Targets: LCP <= 2.5 s, INP <= 200 ms, CLS <= 0.1 (p75). Optimized/responsive images, caching, efficient requests, minimal JS, server-rendered public pages, lazy loading, hero priority. Dashboard: paginated lists, indexed queries, debounced search, efficient aggregation, store-scoped caching, background imports, loading states. SEO: unique titles, meta descriptions, canonicals, Open Graph, product structured data, store sitemaps, human-readable URLs. Do not index auth pages, dashboard, unpublished stores, private previews, checkout sessions, tracking pages. No fabricated ratings/availability metadata.

## 37. LOCALIZATION
Bahasa Indonesia (default) and English. Centralized translation keys; localized UI, validation messages, dates, currency (e.g. Rp245.000), emails; locale-aware numbers; no hardcoded English throughout; natural commerce terminology.

## 38. ANALYTICS
Account funnel: magic_link_requested, magic_link_email_sent, magic_link_verified, authentication_completed, account_created. Onboarding: onboarding_started, store_identity_completed, selling_model_selected, store_created, onboarding_completed, setup_task_completed, first_product_created, first_product_published, storefront_published. Multi-store: additional_store_created, store_switched, store_archived. Shopping: storefront_viewed, collection_viewed, product_viewed, product_added_to_cart, checkout_started, checkout_completed, first_order_received, payment_verified, order_completed. Adoption: catalog_import_started/completed, theme_changed, banner_published, preorder_batch_published/completed, business_rule_published. Properties: event id, timestamp, store id, user id where appropriate, session id, selling model, business category, source, non-sensitive properties. Exclude raw tokens, payment evidence, bank details, unnecessary customer PII. Store activation = published storefront + first valid order; transaction activation = first verified/captured payment; measure separately.

## 39. DEPLOYMENT AND OPS
Environments: local, staging, production; separate config/credentials; no production customer data in dev without approved process. VPS: reverse proxy, TLS, app processes/containers, PostgreSQL, background worker, persistent storage, automated backups, health checks; Docker Compose if appropriate; no unnecessary orchestration. Monitoring: health checks, structured error logging, DB connectivity, job monitoring, email delivery monitoring, import monitoring, disk/storage monitoring, failed transaction visibility; no public internal monitoring without authorization. Backups: scheduled PostgreSQL backups, object storage recovery, secure storage, configurable retention, periodic restore tests (a backup isn't reliable until restored). Deployment safety: run tests, validate config, review migrations, confirm backup, validate health, prepare rollback; no automatic destructive migrations.

## 40. ROADMAP
Phase 0 Foundation and Design System: repo structure, Next.js app, Go API, PostgreSQL, migration framework, env config, storage abstraction, design tokens, Inter, reusable components, responsive layouts, component preview page, deployment foundation. Exit: foundation operational, reusable components available.
Phase 1 Auth and Multi-Store Onboarding: Brevo integration, magic links, secure sessions, account creation, first-store onboarding, ownership, store switcher, manage stores, additional stores, account settings, store authorization. Exit: email auth, create first/additional stores, switch securely.
Phase 2 Beautiful Storefront: dashboard foundation, setup checklist, products, variants, inventory, collections, themes, branding, banners, featured products/collections, editor, preview, publication, public browsing. Exit: publish a professional mobile-friendly storefront.
Phase 3 First Transaction: cart, guest checkout, ready-stock orders, reservations, fulfillment settings, manual bank transfer, evidence, verification, order management, customer profiles, tracking, essential emails. Exit: real customer orders, submits evidence, tracks; merchant verifies and completes.
Phase 4 Flexible Commerce: preorder batches, assignment, batch pricing, quotas, auto open/close, preorder checkout, multiple cart groups, consolidated purchasing, allocation, batch progress, handling fees, minimum order, approval, rule preview. Exit: a jastiper completes a full preorder cycle.
Phase 5 Merchant Efficiency: AI PDF import, extraction, image extraction, review, CSV import, customer management improvements, notification center, email reliability, analytics, operational dashboard improvements.
Phase 6 Marketing and Production Readiness: final marketing site, solution pages, approved pricing content, legal pages, accessibility validation, security testing, performance optimization, monitoring, backup recovery validation, E2E regression. Exit: ready for controlled pilots on pikko.site.
Phase 7 Official Go-to-Market (needs explicit approval; do not execute automatically): migration to pikko.id, Brevo sender domain migration, commercial pricing, support processes, production monitoring, approved marketing content, launch checklist.

## 41. POST-MVP (do not implement)
Visual workflow builder, marketing automation, promotional campaigns, discount codes, abandoned cart campaigns, advanced segmentation, quotation management, customer sourcing requests, deposits, partial payments, installments, custom domains, merchant subdomains, staff invitations, advanced roles, store duplication, wholesale, marketplace discovery, native apps, multi-warehouse, advanced shipping integrations, AI shopping assistant, customer portal, app marketplace, advanced reporting, subscription billing.

## 42. ACCEPTANCE SCENARIOS
A new user auth -> one verified account, onboarding, no password. B returning user -> existing account, authorized store, no duplicate. C scanner GET leaves token available. D first store: store, membership, defaults, checklist, dashboard. E multiple stores independent. F switching: all data reflects selected store, other tabs unaffected. G social seller: publish ready-stock store, customer checks out, merchant verifies payment and fulfills. H jastiper: batch, paid orders, consolidated purchasing, record purchased, allocate, fulfill. I closed preorder: backend rejects storefront or direct API checkout. J multiple fulfillment groups separated. K AI import: editable drafts, no fabrication, nothing auto-published. L business rules: correct rule versions applied and stored. M payment verification: status updated, audit record, email queued. N concurrent checkout: no overselling. O tenant isolation: unauthorized access denied. P historical integrity: price change doesn't alter old orders. Q email failure (Brevo outage): order succeeds, job retried. R storefront draft stays private until published. S domain config: staging domain changes all generated links without changing records. T design consistency across surfaces.

## 43. TESTING
Unit: money, rule precedence, token validation, state transitions, inventory reservation, preorder eligibility, purchasing aggregation, customer matching. Integration: auth, store creation, membership, authorization, checkout, payment verification, DB transactions, outbox, Brevo adapter, jobs, imports. E2E (Playwright): magic link auth via safe test mechanism, onboarding, additional store, switching, product publication, storefront customization, ready-stock checkout, preorder checkout, payment verification, tracking, fulfillment; no test auth bypass in production. Visual regression: auth, onboarding, dashboard, switcher, product editor, storefront editor, public storefront, checkout, tracking at mobile and desktop widths. Security regression: cross-store access, expired/reused tokens, session revocation, unauthorized file access, invalid transitions, duplicate submissions, concurrent stock updates, client price manipulation.

## 44. DESIGN QUALITY ACCEPTANCE
Every screen: Inter, approved tokens, consistent spacing, restrained radii, precise alignment, clear hierarchy, accessible contrast, responsive, functional interactions, loading/empty/error states, keyboard accessibility. Compare polish with brex.com. Reject: generic AI gradients, excessive rounded cards, heavy shadows, random colors, inconsistent fonts, unnecessary icons, repetitive decorative panels, unclear CTA hierarchy, placeholder metrics, fake functionality.

## 45. DEFINITION OF DONE
1 functional requirements implemented; 2 FE+BE integrated; 3 DB persistence; 4 authn/authz validated; 5 business logic tested; 6 design system satisfied; 7 responsive verified; 8 loading/empty/error states; 9 relevant tests pass; 10 existing functionality works; 11 sensitive actions audited; 12 docs updated; 13 deployment verification where authorized. A visually complete screen without backend is not complete; an API without UX is not complete.

## 46. OPEN BUSINESS DECISIONS (do not invent)
Subscription pricing, free plan limits, stores per subscription, AI import commercial limits, payment gateway provider and terms, final logo, final legal docs, support procedures, launch date, domain migration date, customer-facing AI agent, advanced fulfillment integrations. Use configurable foundations.

## 47. INITIAL EXECUTION INSTRUCTION
Start with the foundation: 1 inspect repo/infra; 2 create required docs; 3 architecture and repo structure; 4 DB migration approach; 5 design system; 6 reusable component library; 7 env configuration; 8 Brevo provider integration; 9 auth and multi-store DB models; 10 phased implementation checklist. Then proceed through the roadmap in dependency order. Do not skip authentication and tenant isolation. No disconnected placeholder screens. Development-only sample data is OK; never display fabricated production metrics. Maintain REQUIREMENTS_MATRIX and DEVELOPMENT_PROGRESS throughout. Final product enables an independent seller to authenticate with email only, create one or many stores, publish a beautiful storefront, receive ready-stock and preorder orders, verify payments, manage purchasing and fulfillment, keep independent customer records, communicate progress via Brevo, without unnecessary complexity. Direction: Beautiful storefronts. Flexible ordering. Effortless operations. Pikko. Sell your way.

## Delivery instruction (from user, Indonesian)
Deploy on the VPS `ssh -p 2021 ubuntu@43.163.99.95` as separate containers; it MUST run in Docker. SSH key will be provided by the user as a secret (name to be confirmed).
