import { getData, http } from "./http";

export type WSTicketResult = {
  ticket: string;
  expires_in: number;
};

/** 用当前 Cookie 会话换取一次性 WebSocket 握手 ticket（30s 有效，单次使用；URL 仅允许 ?ticket=）。 */
export async function fetchWSTicket(scope?: string): Promise<string> {
  const res = await getData<WSTicketResult>(
    http.post("/auth/ws-ticket", scope ? { scope } : {}, { silentErrorToast: true }),
  );
  return res.ticket;
}

/** 构建带 ticket 的 WebSocket URL（不在 query 中携带 JWT）。 */
export async function buildAuthenticatedWSURL(
  path: string,
  params: Record<string, string | number | undefined> = {},
  scope?: string,
): Promise<string> {
  const ticket = await fetchWSTicket(scope);
  const url = new URL(path, window.location.origin);
  url.protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === "") continue;
    url.searchParams.set(key, String(value));
  }
  url.searchParams.set("ticket", ticket);
  return url.toString();
}

export async function openAuthenticatedWebSocket(
  path: string,
  params: Record<string, string | number | undefined> = {},
  scope?: string,
): Promise<WebSocket> {
  const url = await buildAuthenticatedWSURL(path, params, scope);
  return new WebSocket(url);
}
