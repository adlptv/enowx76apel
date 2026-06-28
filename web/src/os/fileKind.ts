const imageExt = ["png", "jpg", "jpeg", "gif", "webp", "svg", "bmp", "ico", "avif"];

// fileKind maps a filename to how it should be previewed.
export function fileKind(name: string): "text" | "image" {
  const ext = name.split(".").pop()?.toLowerCase() ?? "";
  return imageExt.includes(ext) ? "image" : "text";
}
