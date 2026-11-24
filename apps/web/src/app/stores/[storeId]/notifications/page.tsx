import { redirect } from "next/navigation";
import { getSession, serverFetch } from "@/lib/server-api";
import { urls } from "@/lib/urls";
import { NotificationsClient } from "@/components/notifications/notifications-client";

type Notification = {
  id: string;
  type: string;
  title: string;
  body?: string | null;
  resource_type?: string | null;
  resource_id?: string | null;
  read_at?: string | null;
  created_at: string;
};

export default async function NotificationsPage({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());

  const data = await serverFetch<{
    notifications: Notification[];
    total: number;
    unread_count: number;
  }>(`/api/v1/stores/${storeId}/notifications?limit=50`);

  return (
    <div className="mx-auto max-w-dashboard">
      <h1 className="text-[30px] font-medium text-foreground">Notifications</h1>
      <p className="mt-2 text-sm text-foreground-secondary">
        Store-scoped operational alerts. Unread: {data?.unread_count ?? 0}.
      </p>
      <NotificationsClient
        storeId={storeId}
        initial={data?.notifications || []}
        unreadCount={data?.unread_count ?? 0}
      />
    </div>
  );
}
