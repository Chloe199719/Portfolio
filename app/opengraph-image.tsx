import { ImageResponse } from "next/og";
export const alt =
  "Chloe Pratas — Software, side quests, and everything in between.";
export const size = { width: 1200, height: 630 };
export const contentType = "image/png";
export default function Image() {
  return new ImageResponse(
    <div
      style={{
        width: "100%",
        height: "100%",
        display: "flex",
        flexDirection: "column",
        justifyContent: "space-between",
        padding: "65px 80px",
        background: "#191516",
        color: "#f5eeee",
        fontFamily: "sans-serif",
      }}
    >
      <div style={{ display: "flex", fontSize: 55 }}>
        chloe<span style={{ color: "#e76a6a" }}>.</span>
      </div>
      <div
        style={{
          display: "flex",
          flexDirection: "column",
          fontSize: 76,
          lineHeight: 1.08,
        }}
      >
        <span>Software, side quests,</span>
        <span style={{ color: "#e76a6a" }}>and everything in between.</span>
      </div>
      <div style={{ display: "flex", fontSize: 21, color: "#bba5a7" }}>
        Chloe Pratas · Software engineer & curious human
      </div>
    </div>,
    size,
  );
}
