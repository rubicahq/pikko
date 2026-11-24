import { redirect } from "next/navigation";
import { getOnboardingDraft, getSession, getStores } from "@/lib/server-api";
import { OnboardingWizard } from "@/components/onboarding/wizard";
import { storefrontHost } from "@/lib/site";
import { urls } from "@/lib/urls";

export default async function OnboardingPage() {
  const session = await getSession();
  if (!session) redirect(urls.auth());

  const stores = await getStores();
  const draft = await getOnboardingDraft();
  if (!draft) redirect(urls.stores());

  // If user somehow landed here with stores and no incomplete intent, allow additional via /stores/new
  return (
    <OnboardingWizard
      draft={draft}
      hasExistingStores={stores.length > 0}
      storefrontBase={storefrontHost()}
    />
  );
}
