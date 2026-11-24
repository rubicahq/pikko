"use client";

import { useCallback, useEffect, useState } from "react";
import { apiFetch, ApiError } from "@/lib/api";
import { ROLE_LABELS, hasPermission } from "@/lib/rbac";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { PageHeader } from "@/components/ui/page-header";
import { EmptyState } from "@/components/ui/empty-state";
import { StatusBadge } from "@/components/ui/status-badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

type Member = {
  id: string;
  user_id: string;
  email: string;
  display_name: string | null;
  role: string;
  status: string;
  created_at: string;
};

type Invite = {
  id: string;
  email: string;
  role: string;
  status: string;
  expires_at: string;
  last_sent_at: string;
  created_at: string;
  dev_token?: string;
};

export function TeamPanel({ storeId }: { storeId: string }) {
  const [members, setMembers] = useState<Member[]>([]);
  const [invites, setInvites] = useState<Invite[]>([]);
  const [roles, setRoles] = useState<string[]>([]);
  const [permissions, setPermissions] = useState<string[]>([]);
  const [myRole, setMyRole] = useState<string>("");
  const [email, setEmail] = useState("");
  const [role, setRole] = useState("cs");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [busyId, setBusyId] = useState<string | null>(null);

  const load = useCallback(async () => {
    const [membersRes, invitesRes] = await Promise.all([
      apiFetch<{
        members: Member[];
        roles: string[];
        permissions: string[];
        my_role: string;
      }>(`/api/v1/stores/${storeId}/team/members`),
      apiFetch<{ invites: Invite[] }>(`/api/v1/stores/${storeId}/team/invites`),
    ]);
    setMembers(membersRes.members || []);
    setRoles(membersRes.roles || []);
    setPermissions(membersRes.permissions || []);
    setMyRole(membersRes.my_role || "");
    setInvites(invitesRes.invites || []);
    setRole((current) => current || membersRes.roles?.[0] || "cs");
  }, [storeId]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        await load();
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof ApiError ? err.message : "Unable to load team.");
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [load]);

  async function invite() {
    setLoading(true);
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/team/invites`, {
        body: { email, role },
      });
      setEmail("");
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to send invite.");
    } finally {
      setLoading(false);
    }
  }

  async function resend(id: string) {
    setBusyId(id);
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/team/invites/${id}/resend`, { method: "POST" });
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to resend.");
    } finally {
      setBusyId(null);
    }
  }

  async function cancel(id: string) {
    setBusyId(id);
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/team/invites/${id}/cancel`, { method: "POST" });
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to cancel.");
    } finally {
      setBusyId(null);
    }
  }

  async function revoke(id: string) {
    setBusyId(id);
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/team/members/${id}/revoke`, { method: "POST" });
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to revoke.");
    } finally {
      setBusyId(null);
    }
  }

  const canInvite = hasPermission(permissions, "team.invite");
  const canManage = hasPermission(permissions, "team.manage");
  const pending = invites.filter((i) => i.status === "pending");

  return (
    <div className="mx-auto max-w-dashboard space-y-8">
      <PageHeader
        title="Team"
        description={`Manage who can access this store. You are signed in as ${ROLE_LABELS[myRole] || myRole || "member"}.`}
      />

      {error ? <p className="text-sm text-error">{error}</p> : null}

      {canInvite ? (
        <section className="rounded-lg border border-border p-5">
          <h2 className="text-base font-medium text-foreground">Invite teammate</h2>
          <p className="mt-1 text-sm text-foreground-secondary">
            They receive a Brevo email with a magic link to join this store only.
          </p>
          <div className="mt-4 flex flex-col gap-3 md:flex-row md:items-end">
            <label className="flex-1 text-sm">
              <span className="mb-1.5 block text-foreground-secondary">Email</span>
              <Input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="teammate@example.com"
              />
            </label>
            <label className="md:w-48 text-sm">
              <span className="mb-1.5 block text-foreground-secondary">Role</span>
              <select
                className="flex h-10 w-full rounded-md border border-border bg-background px-3 text-sm"
                value={role}
                onChange={(e) => setRole(e.target.value)}
              >
                {(roles.length ? roles : Object.keys(ROLE_LABELS).filter((r) => r !== "owner")).map(
                  (r) => (
                    <option key={r} value={r}>
                      {ROLE_LABELS[r] || r}
                    </option>
                  ),
                )}
              </select>
            </label>
            <Button loading={loading} onClick={invite} disabled={!email.trim()}>
              Send invite
            </Button>
          </div>
        </section>
      ) : null}

      <section>
        <h2 className="mb-3 text-base font-medium text-foreground">Members</h2>
        {members.length === 0 ? (
          <EmptyState title="No members" description="Invite teammates to collaborate." />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Person</TableHead>
                <TableHead>Role</TableHead>
                <TableHead className="w-32" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {members.map((m) => (
                <TableRow key={m.id}>
                  <TableCell>
                    <div className="font-medium text-foreground">{m.display_name || m.email}</div>
                    {m.display_name ? (
                      <div className="text-caption text-foreground-muted">{m.email}</div>
                    ) : null}
                  </TableCell>
                  <TableCell>{ROLE_LABELS[m.role] || m.role}</TableCell>
                  <TableCell className="text-right">
                    {canManage && m.role !== "owner" ? (
                      <Button
                        variant="outline"
                        size="sm"
                        loading={busyId === m.id}
                        onClick={() => revoke(m.id)}
                      >
                        Revoke
                      </Button>
                    ) : null}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </section>

      <section>
        <h2 className="mb-3 text-base font-medium text-foreground">Pending invites</h2>
        {pending.length === 0 ? (
          <p className="text-sm text-foreground-secondary">No pending invites.</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Email</TableHead>
                <TableHead>Role</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="w-48" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {pending.map((inv) => (
                <TableRow key={inv.id}>
                  <TableCell>{inv.email}</TableCell>
                  <TableCell>{ROLE_LABELS[inv.role] || inv.role}</TableCell>
                  <TableCell>
                    <StatusBadge status="pending" label="Pending" />
                  </TableCell>
                  <TableCell className="space-x-2 text-right">
                    {canInvite ? (
                      <Button
                        variant="outline"
                        size="sm"
                        loading={busyId === inv.id}
                        onClick={() => resend(inv.id)}
                      >
                        Resend
                      </Button>
                    ) : null}
                    {canManage ? (
                      <Button
                        variant="outline"
                        size="sm"
                        loading={busyId === inv.id}
                        onClick={() => cancel(inv.id)}
                      >
                        Cancel
                      </Button>
                    ) : null}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </section>
    </div>
  );
}
