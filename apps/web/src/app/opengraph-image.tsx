import { ImageResponse } from "next/og";
import { appBaseDomain } from "@/lib/site";

export const runtime = "edge";
export const alt = "Pikko — Sell your way.";
export const size = { width: 1200, height: 630 };
export const contentType = "image/png";

export default function OpenGraphImage() {
  const domain = appBaseDomain() === "localhost" ? "pikko.id" : appBaseDomain();
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
        <div style={{ display: "flex", alignItems: "center", gap: 16 }}>
          <div
            style={{
              width: 48,
              height: 48,
              background: "#FF3D00",
              borderRadius: 12,
            }}
          />
          <div style={{ fontSize: 36, fontWeight: 500, letterSpacing: "-0.03em" }}>Pikko</div>
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
