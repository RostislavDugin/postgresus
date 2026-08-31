// Sponsorship configuration — shared by the server page (to render the tier cards)
// and by PaddleInitComponent (to initialise Paddle.js).
//
// PADDLE_CLIENT_TOKEN is the Paddle Billing *client-side* token ("live_" for
// production, "test_" for sandbox). Each tier carries one or two recurring Paddle
// Price IDs: a monthly price and, for the larger tiers, an annual price (≈ 10×
// monthly, i.e. two months free). Tiers without an annual price simply omit it.
//
// If a priceId is left empty, that button gracefully falls back to the Telegram
// contact instead of opening a broken checkout.

export const PADDLE_CLIENT_TOKEN = "live_11a32cf90c4e8ce9ea1f6aa6b29";
export const PADDLE_SANDBOX = false;

/** Telegram contact used for "Message me" links and the no-Paddle fallback. */
export const TELEGRAM_URL = "https://t.me/rostislav_dugin";

/** A single billing option: its display price and the recurring Paddle Price ID. */
export interface BillingPrice {
  /** Display price, e.g. "$250". */
  price: string;
  /** Recurring Paddle Price ID. Leave "" to fall back to the Telegram contact. */
  priceId: string;
}

export interface Tier {
  /** Tier name shown on the card. */
  name: string;
  /** Monthly billing option (every tier has one). */
  monthly: BillingPrice;
  /** Annual billing option — omitted for tiers without an annual price. */
  annual?: BillingPrice;
  /** Highlight this tier as the recommended one. */
  popular?: boolean;
}

export const TIERS: Tier[] = [
  { name: "Supporter", monthly: { price: "$25", priceId: "pri_01ktqst3kr153241zp1htecz9w" } },
  { name: "Backer", monthly: { price: "$100", priceId: "pri_01ktqsvm5wf86ext20jawzdqj7" } },
  {
    name: "Startup",
    monthly: { price: "$250", priceId: "pri_01ktqswk7dysf5dc1mcnk3vyrq" },
    annual: { price: "$2,500", priceId: "pri_01ktqt2atfh2n72teyz2egd0nk" },
    popular: true,
  },
  {
    name: "Growth",
    monthly: { price: "$500", priceId: "pri_01ktqsxgzdfx4z0p1xjzbgg38e" },
    annual: { price: "$5,000", priceId: "pri_01ktqt3mnyfy6qym41te5cdek6" },
  },
  {
    name: "Business",
    monthly: { price: "$1,000", priceId: "pri_01ktqsyqc609gsyahva5sskntn" },
    annual: { price: "$10,000", priceId: "pri_01ktqt4xjjahsjz4vkfv4fd9w4" },
  },
  {
    name: "Scale",
    monthly: { price: "$2,500", priceId: "pri_01ktqszn1hep5xq3b46xpd97nk" },
    annual: { price: "$25,000", priceId: "pri_01ktqt630kgax3v2z3kqykb6cf" },
  },
  {
    name: "Enterprise",
    monthly: { price: "$5,000", priceId: "pri_01ktqt0wn9c2szbmsks6f6xa72" },
    annual: { price: "$50,000", priceId: "pri_01ktqt7atzxdjm55rhrewxkqg5" },
  },
];

/** True once a client token is configured — gates loading Paddle.js at all. */
export const PADDLE_ENABLED = PADDLE_CLIENT_TOKEN.length > 0;

/** A price opens the Paddle overlay only when both the token and its price ID exist. */
export function priceUsesPaddle(priceId: string): boolean {
  return PADDLE_ENABLED && priceId.length > 0;
}
