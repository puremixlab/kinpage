import { httpsURL, MobileError, requestText } from './network.ts';

export type Connection = { origin: string; username: string; password: string };
export type DesktopSite = { id: string; name: string };
export type Publication = { articleId: string; cid: string };

export const publicGateway = 'https://eth.sucks';
const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
const cidPattern = /^[a-z0-9]{32,120}$/i;

export function normalizeConnection(origin: string, username: string, password: string): Connection {
  const url = httpsURL(origin);
  if (!url || url.pathname !== '/' || url.search || url.hash || url.origin.length > 255) {
    throw new MobileError('invalid_desktop_url');
  }
  const account = username.trim();
  if (!account || account.length > 80 || !password || password.length > 256 || /[:\x00-\x1f\x7f]/.test(account)) {
    throw new MobileError('invalid_credentials');
  }
  return { origin: url.origin, username: account, password };
}

async function desktopRequest(
  connection: Connection,
  path: string,
  fetcher: typeof fetch,
  article?: { title: string; content: string },
): Promise<unknown> {
  const credentials = new TextEncoder().encode(connection.username + ':' + connection.password);
  const { response, text } = await requestText(
    fetcher,
    connection.origin + '/v0/mobile/' + path,
    {
      method: article ? 'PUT' : 'GET',
      redirect: 'error',
      headers: {
        Accept: 'application/json',
        Authorization: 'Basic ' + btoa(String.fromCharCode(...credentials)),
        ...(article && { 'Content-Type': 'application/json' }),
      },
      body: article ? JSON.stringify(article) : undefined,
    },
    256 * 1024,
    article ? 120_000 : 12_000,
  );
  if (!response.ok) {
    switch (response.status) {
      case 401: throw new MobileError('auth_required');
      case 403: throw new MobileError('auth_disabled');
      case 404: throw new MobileError(article ? 'site_unavailable' : 'desktop_unsupported');
      case 409: throw new MobileError('article_conflict');
      case 502: throw new MobileError('publish_failed');
      default: throw new MobileError('desktop_error');
    }
  }
  try {
    return JSON.parse(text);
  } catch {
    throw new MobileError('desktop_unsupported');
  }
}

export async function listDesktopSites(connection: Connection, fetcher: typeof fetch = fetch): Promise<DesktopSite[]> {
  const sites = await desktopRequest(connection, 'sites', fetcher);
  if (!Array.isArray(sites) || sites.some((site) =>
    !site || typeof site.id !== 'string' || !uuidPattern.test(site.id) || typeof site.name !== 'string'
  )) {
    throw new MobileError('desktop_unsupported');
  }
  return sites.map(({ id, name }) => ({ id, name }));
}

export async function publishArticle(
  connection: Connection,
  siteId: string,
  article: { id: string; title: string; content: string },
  fetcher: typeof fetch = fetch,
): Promise<Publication> {
  if (!uuidPattern.test(siteId) || !uuidPattern.test(article.id)) throw new MobileError('desktop_error');
  const path = 'sites/' + siteId + '/articles/' + article.id;
  const receipt = await desktopRequest(connection, path, fetcher, {
    title: article.title.trim(),
    content: article.content,
  }) as Partial<Publication> | null;
  if (!receipt || typeof receipt.articleId !== 'string' ||
    receipt.articleId.toLowerCase() !== article.id.toLowerCase() ||
    typeof receipt.cid !== 'string' || !cidPattern.test(receipt.cid)) {
    throw new MobileError('desktop_error');
  }
  return { articleId: receipt.articleId, cid: receipt.cid };
}

export function publishedURL(cid: string, articleId: string): string {
  return publicGateway + '/ipfs/' + cid + '/' + articleId.toUpperCase() + '/';
}
