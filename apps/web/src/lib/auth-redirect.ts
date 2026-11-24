import { redirect } from "next/navigation";
import { getSession } from "@/lib/server-api";
import { urls } from "@/lib/urls";

/** If a valid session cookie is present, send the user to the dashboard. */
export async function redirectIfAuthenticated(): Promise<void> {
  const session = await getSession();
  if (!session) return;
  redirect(urls.dashboardAbsolute(urls.stores()));
}
