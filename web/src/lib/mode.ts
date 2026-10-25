const mode =
  import.meta.env.VITE_API_MODE ?? (import.meta.env.MODE === "real" ? "real" : "mock");

export const isMock = mode !== "real";
