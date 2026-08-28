export function extractHostPort(link: string): { host: string; port: number } | null {
  if (!link) return null;
  link = link.trim();

  if (link.startsWith("vmess://")) {
    try {
      const raw = link.slice(8);
      const decoded = atob(raw);
      const json = JSON.parse(decoded);
      return { host: json.add, port: parseInt(json.port) };
    } catch {
      return null;
    }
  }

  try {
    const url = new URL(link);
    const host = url.hostname;
    const port = parseInt(url.port);
    if (host && port) {
      return { host, port };
    }
  } catch {}

  // Fallback string parsing
  try {
    let rest = link;
    if (rest.includes("://")) {
      rest = rest.split("://")[1];
    }
    if (rest.includes("@")) {
      rest = rest.split("@")[1];
    }
    if (rest.includes("#")) {
      rest = rest.split("#")[0];
    }
    if (rest.includes("?")) {
      rest = rest.split("?")[0];
    }
    rest = rest.split("/")[0];
    if (rest.includes(":")) {
      const parts = rest.split(":");
      const port = parseInt(parts[parts.length - 1]);
      const host = parts.slice(0, -1).join(":").replace(/[\[\]]/g, "");
      if (host && !isNaN(port)) return { host, port };
    }
  } catch {}
  return null;
}

export const pingLocalNode = async (server: string, port: number, timeoutMs = 2000): Promise<number> => {
  const start = performance.now();
  const controller = new AbortController();
  const id = setTimeout(() => controller.abort(), timeoutMs);

  try {
    await fetch(`https://${server}:${port}`, {
      mode: 'no-cors',
      cache: 'no-cache',
      signal: controller.signal
    });
    clearTimeout(id);
    return Math.round(performance.now() - start);
  } catch (err: any) {
    clearTimeout(id);
    if (err.name === 'AbortError') {
      return -1; // Timeout
    }
    return Math.round(performance.now() - start);
  }
};

export const testLocalAll = async (
  nodes: Array<{ server?: string; port?: number; link?: string; rtt: number }>,
  onUpdate: (index: number, rtt: number) => void
) => {
  const poolLimit = 6;

  const targets = nodes.map((n, index) => {
    let host = n.server || "";
    let port = n.port || 0;
    if ((!host || !port) && n.link) {
      const hp = extractHostPort(n.link);
      if (hp) {
        host = hp.host;
        port = hp.port;
      }
    }
    return { host, port, index, rtt: n.rtt };
  }).filter(t => t.host && t.port && t.rtt !== -1);

  let activeIndex = 0;
  const runNext = async () => {
    while (activeIndex < targets.length) {
      const t = targets[activeIndex++];
      onUpdate(t.index, -2); // -2 represents "testing..."
      const rtt = await pingLocalNode(t.host, t.port);
      onUpdate(t.index, rtt);
    }
  };

  const promises = [];
  for (let i = 0; i < Math.min(poolLimit, targets.length); i++) {
    promises.push(runNext());
  }
  await Promise.all(promises);
};