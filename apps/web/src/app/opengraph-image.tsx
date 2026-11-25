/* ImageResponse requires raw <img>; next/image is not supported here. */
/* eslint-disable @next/next/no-img-element */
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { ImageResponse } from "next/og";
import { appBaseDomain } from "@/lib/site";

export const runtime = "nodejs";
export const alt = "Pikko — Sell your way.";
export const size = { width: 1200, height: 630 };
export const contentType = "image/png";

export default async function OpenGraphImage() {
  const domain = appBaseDomain() === "localhost" ? "pikko.id" : appBaseDomain();
  const [iconBuf, logoBuf] = await Promise.all([
    readFile(join(process.cwd(), "public/brand/pikko-icon.png")),
    readFile(join(process.cwd(), "public/brand/pikko-logo.png")),
  ]);
  const iconSrc = `data:image/png;base64,${iconBuf.toString("base64")}`;
  const logoSrc = `data:image/png;base64,${logoBuf.toString("base64")}`;

  return new ImageResponse(
    (
      <div
        style={{
          width: "100%",
          height: "100%",
          display: "flex",
          flexDirection: "column",
          justifyContent: "space-between",
          background: "#000710",
          color: "#FCFCFD",
          padding: "72px 80px",
          fontFamily: "Arial, sans-serif",
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: 20 }}>
          <img src={iconSrc} width={56} height={56} alt="" />
          <img src={logoSrc} width={108} height={44} alt="Pikko" />
        </div>
        <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
          <div
            style={{
              fontSize: 72,
              fontWeight: 500,
              letterSpacing: "-0.04em",
              lineHeight: 1.02,
              maxWidth: 900,
            }}
          >
            Sell your way.
          </div>
          <div style={{ fontSize: 28, color: "#B9BBC6", maxWidth: 720, lineHeight: 1.4 }}>
            Beautiful storefronts. Flexible ordering. Effortless operations.
          </div>
        </div>
        <div style={{ fontSize: 22, color: "#8B8D98" }}>{domain}</div>
      </div>
    ),
    { ...size },
  );
}
