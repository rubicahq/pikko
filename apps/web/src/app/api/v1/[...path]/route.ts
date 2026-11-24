import { NextRequest, NextResponse } from "next/server";

const upstream =
  process.env.API_BASE_URL || process.env.NEXT_PUBLIC_API_BASE_URL || "http://127.0.0.1:8080";

async function proxy(req: NextRequest, path: string[]) {
  const target = `${upstream}/api/v1/${path.join("/")}${req.nextUrl.search}`;
  const headers = new Headers();
  const pass = ["content-type", "authorization", "cookie", "x-request-id", "x-forwarded-for"];
  for (const key of pass) {
    const v = req.headers.get(key);
    if (v) headers.set(key, v);
  }
  headers.set("x-forwarded-proto", req.nextUrl.protocol.replace(":", ""));
  if (!headers.has("x-forwarded-for")) {
    headers.set("x-forwarded-for", req.headers.get("x-real-ip") || "127.0.0.1");
  }

  const init: RequestInit = {
    method: req.method,
    headers,
    redirect: "manual",
  };
  if (req.method !== "GET" && req.method !== "HEAD") {
    init.body = await req.arrayBuffer();
  }

  const res = await fetch(target, init);
  const outHeaders = new Headers();
  res.headers.forEach((value, key) => {
    const k = key.toLowerCase();
    if (k === "transfer-encoding" || k === "content-encoding") return;
    if (k === "set-cookie") return; // handled below
    outHeaders.set(key, value);
  });

  const response = new NextResponse(await res.arrayBuffer(), {
    status: res.status,
    headers: outHeaders,
  });

  const getSetCookie = (res.headers as Headers & { getSetCookie?: () => string[] }).getSetCookie?.();
  if (getSetCookie && getSetCookie.length) {
    for (const c of getSetCookie) {
      response.headers.append("set-cookie", c);
    }
  } else {
    const single = res.headers.get("set-cookie");
    if (single) response.headers.append("set-cookie", single);
  }

  return response;
}

type Ctx = { params: Promise<{ path: string[] }> };

export async function GET(req: NextRequest, ctx: Ctx) {
  const { path } = await ctx.params;
  return proxy(req, path);
}
export async function POST(req: NextRequest, ctx: Ctx) {
  const { path } = await ctx.params;
  return proxy(req, path);
}
export async function PATCH(req: NextRequest, ctx: Ctx) {
  const { path } = await ctx.params;
  return proxy(req, path);
}
export async function PUT(req: NextRequest, ctx: Ctx) {
  const { path } = await ctx.params;
  return proxy(req, path);
}
export async function DELETE(req: NextRequest, ctx: Ctx) {
  const { path } = await ctx.params;
  return proxy(req, path);
}
export async function OPTIONS(req: NextRequest, ctx: Ctx) {
  const { path } = await ctx.params;
  return proxy(req, path);
}
