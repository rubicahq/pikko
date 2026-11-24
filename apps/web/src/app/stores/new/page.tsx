import { redirect } from "next/navigation";
import { getOnboardingDraft, getSession, getStores } from "@/lib/server-api";
import { OnboardingWizard } from "@/components/onboarding/wizard";
import { storefrontHost } from "@/lib/site";
import { urls } from "@/lib/urls";

export default async function NewStorePage() {
  const session = await getSession();
  if (!session) redirect(urls.auth());

  const stores = await getStores();
  // Reset/ensure draft for additional store
  const draft = await getOnboardingDraft();
  if (!draft) redirect(urls.stores());

  return (
    <OnboardingWizard
      draft={{ ...draft, is_first_store: stores.length === 0 }}
      hasExistingStores={stores.length > 0}
      storefrontBase={storefrontHost()}
    />
  );
}
