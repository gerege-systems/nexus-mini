// Админ аппын API клиент — same-origin /api (rewrites → Go).

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: { "Content-Type": "application/json", ...init?.headers },
    credentials: "same-origin",
  });
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
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

export type Me = {
  user: { id: string; name: string; email: string; platform_admin: boolean };
};
