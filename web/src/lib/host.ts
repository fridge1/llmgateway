const IMAGE_HOSTS = new Set(["image.your-domain.com", "image.localhost"]);

export function isImageHost(): boolean {
  if (typeof window === "undefined") return false;
  return IMAGE_HOSTS.has(window.location.hostname);
}
