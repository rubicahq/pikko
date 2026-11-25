import { PricingPlans } from "@/components/marketing/pricing-plans";
import { marketingMetadata } from "@/lib/marketing-seo";

export const metadata = marketingMetadata({
  title: "Pricing",
  description:
    "Pikko store plans — Basic, Standard, Premium, and Ultimate. IDR pricing with annual savings.",
  path: "/pricing",
});

export default function PricingPage() {
  return (
    <div className="mx-auto max-w-marketing px-4 py-16 md:px-8 md:py-24">
      <PricingPlans />
    </div>
  );
}
