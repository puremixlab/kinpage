export class MobileError extends Error {
  readonly code: string;

  constructor(code: string) {
    super(code);
    this.code = code;
  }
}

export function httpsURL(value: string, base?: string): URL | null {
  if (!value.trim() || /[\u0000-\u001f\u007f\\]/.test(value)) return null;
  try {
    const url = new URL(value.trim(), base);
    if (url.protocol === 'https:' && !url.username && !url.password) return url;
  } catch {
    return null;
  }
  return null;
}

export async function requestText(
  fetcher: typeof fetch,
  url: string,
  init: RequestInit,
  maxBytes: number,
  timeoutMs: number,
) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    const response = await fetcher(url, {
      credentials: 'omit',
      ...init,
      signal: controller.signal,
    });
    if (!response.ok) return { response, text: '' };
    const declaredSize = Number(response.headers.get('content-length'));
    if (declaredSize > maxBytes) throw new MobileError('response_too_large');
    if (!response.body) return { response, text: '' };

    const reader = response.body.getReader();
    const chunks: Uint8Array[] = [];
    let size = 0;
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      size += value.byteLength;
      if (size > maxBytes) throw new MobileError('response_too_large');
      chunks.push(value);
    }
    const data = new Uint8Array(size);
    let offset = 0;
    for (const chunk of chunks) {
      data.set(chunk, offset);
      offset += chunk.byteLength;
    }
    return { response, text: new TextDecoder().decode(data).replace(/^\uFEFF/, '') };
  } catch (error) {
    if (error instanceof MobileError) throw error;
    throw new MobileError(controller.signal.aborted ? 'timeout' : 'network');
  } finally {
    clearTimeout(timer);
    controller.abort();
  }
}
