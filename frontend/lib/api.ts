// API клиент — бүх хүсэлт same-origin /api (next.config rewrites → Go).

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const lang =
    typeof window !== "undefined" ? localStorage.getItem("nexus_locale") || "mn" : "mn";
  const res = await fetch(path, {
    ...init,
    headers: { "Content-Type": "application/json", "X-Lang": lang, ...init?.headers },
    credentials: "same-origin",
  });
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    // Session дууссан бол хаа сайгүй "loading forever" болгохгүй —
    // төвлөрсөн байдлаар нэвтрэх хуудас руу.
    if (
      res.status === 401 &&
      path !== "/api/login" &&
      !window.location.pathname.startsWith("/login")
    ) {
      window.location.href = "/login";
    }
    throw new ApiError(res.status, (body as { error?: string }).error || res.statusText);
  }
  return body as T;
}

export const api = {
  get: <T>(path: string) => req<T>(path),
  post: <T>(path: string, data?: unknown) =>
    req<T>(path, { method: "POST", body: data === undefined ? undefined : JSON.stringify(data) }),
  put: <T>(path: string, data?: unknown) =>
    req<T>(path, { method: "PUT", body: JSON.stringify(data) }),
  del: <T>(path: string) => req<T>(path, { method: "DELETE" }),
};

// ─── Төрлүүд ───

export type Me = {
  user: { id: string; name: string; email: string; platform_admin: boolean };
  tenant_id: string;
  tenants: { id: string; slug: string; name: string }[];
  permissions: Record<string, "all" | "own">;
};

export type StoreApp = {
  id: string;
  short_id: string;
  name: string;
  version: string;
  description: string;
  publisher: string;
  go_module: string;
  compiled: boolean;
  status: "" | "enabled" | "disabled";
  installed_version: string;
};

export type MenuApp = {
  app_id: string;
  short_id: string;
  name: string;
  items: { id: string; label: string; path: string; icon: string; order: number }[];
};

export type Role = {
  id: string;
  code: string;
  name: string;
  implies: string;
  active: boolean;
  grants: Record<string, "all" | "own">;
};

export type Permission = {
  code: string;
  module_id: string;
  name: string;
  description: string;
  own_scope: boolean;
};

export type Member = {
  membership_id: string;
  user_id: string;
  name: string;
  email: string;
  roles: string[];
};

export type AuditEntry = {
  id: number;
  user_id: string;
  user_name: string;
  action: string;
  object: string;
  details: Record<string, unknown>;
  hash: string;
  occurred_at: string;
};

export type Device = {
  id: string;
  name: string;
  kind: string;
  serial: string;
  status: "active" | "repair" | "lost" | "retired";
  note: string;
  created_by: string;
  owner_name: string;
  created_at: string;
};
