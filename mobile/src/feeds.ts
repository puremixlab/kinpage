import { publicGateway } from './desktop.ts';
import { discoverFeedURLs, parseFeed } from './feed-format.ts';
import { httpsURL, MobileError, requestText } from './network.ts';

export function subscriptionURL(raw: string): string {
  const value = raw.trim();
  const isENS = /^[a-z0-9-]+(?:\.[a-z0-9-]+)*\.eth$/i;
  const decentralized = /^(ipfs|ipns|planet):\/\/([a-z0-9.-]+)(\/[^?#]*)?$/i.exec(value);
  if (decentralized) {
    const [, protocol, name, path = ''] = decentralized;
    const root = protocol.toLowerCase() === 'planet' && isENS.test(name)
      ? 'https://' + name + '.limo' : publicGateway + '/' + (protocol.toLowerCase() === 'ipfs' ? 'ipfs' : 'ipns') + '/' + name;
    const url = httpsURL(root + path);
    if (!url) throw new MobileError('invalid_feed_url');
    if (!url.pathname.endsWith('/planet.json') && !/\.(xml|json|rss|atom)$/i.test(url.pathname)) {
      url.pathname = url.pathname.replace(/\/$/, '') + '/planet.json';
    }
    return url.href;
  }
  const url = httpsURL(isENS.test(value) ? 'https://' + value + '.limo/' : value);
  if (!url) throw new MobileError('invalid_feed_url');
  url.hash = '';
  return url.href;
}

async function download(url: string, fetcher: typeof fetch) {
  const headers = { Accept: 'application/rss+xml, application/atom+xml, application/feed+json, application/json, text/html, */*' };
  for (let redirects = 0; redirects <= 4; redirects++) {
    const { response, text } = await requestText(fetcher, url, { headers, redirect: 'manual' }, 4 * 1024 * 1024, 15_000);
    if (response.ok) return { text, url };
    const next = httpsURL(response.headers.get('location') ?? '', url);
    if (![301, 302, 303, 307, 308].includes(response.status) || !next) throw new MobileError('feed_http');
    next.hash = '';
    url = next.href;
  }
  throw new MobileError('feed_http');
}

export async function loadFeed(raw: string, fetcher: typeof fetch = fetch, discover = true) {
  const first = await download(subscriptionURL(raw), fetcher);
  const feed = parseFeed(first.text, first.url);
  if (feed) return { ...feed, url: first.url };
  if (discover && /<html[\s>]|<!doctype html|<link[\s>]/i.test(first.text.slice(0, 2048))) {
    const home = new URL(first.url);
    home.hash = '';
    home.search = '';
    if (!/\.[a-z0-9]{1,8}$/i.test(home.pathname) && !home.pathname.endsWith('/')) home.pathname += '/';
    const alternatives = [
      ...discoverFeedURLs(first.text, first.url).slice(0, 2),
      new URL('planet.json', home).href,
    ];
    for (const url of new Set(alternatives)) {
      if (url === first.url) continue;
      try {
        const candidate = await download(url, fetcher);
        const result = parseFeed(candidate.text, candidate.url);
        if (result) return { ...result, url: candidate.url };
      } catch {
        // A page can advertise several feeds, some of which may be unavailable.
      }
    }
  }
  throw new MobileError('feed_format');
}
