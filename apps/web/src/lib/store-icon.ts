import type { Store } from "@/lib/api";
import { urls } from "@/lib/urls";

/** Same-origin store icon URL for dashboard UI; undefined when no icon is saved. */
export function storeIconSrc(
  store: Pick<Store, "id" | "store_icon_file_id" | "store_icon_url">,
): string | undefined {
  if (store.store_icon_file_id) {
    return urls.file(store.id, store.store_icon_file_id);
  }
  if (store.store_icon_url?.startsWith("/")) {
    return store.store_icon_url;
  }
  return undefined;
}

export function storeNameInitial(name: string) {
  return (name.trim()[0] || "S").toUpperCase();
}
