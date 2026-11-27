import { DomUtils, parseDocument, Parser } from 'htmlparser2';
import { httpsURL } from './network.ts';

export type FeedEntry = {
  id: string;
  title: string;
  content: string;
  format: 'text' | 'markdown';
  url: string;
  publishedAt: number | null;
};

export type Feed = { title: string; entries: FeedEntry[] };
type Node = ReturnType<typeof parseDocument>['children'][number];
type Element = Extract<Node, { attribs: object }>;

const maxEntries = 100;
const maxArticleLength = 60_000;
const hiddenTags = new Set(['script', 'style', 'head', 'template', 'noscript', 'svg']);
const blockTags = new Set(['p', 'div', 'section', 'article', 'li', 'blockquote', 'pre', 'h1', 'h2', 'h3', 'h4', 'br']);

export function htmlText(html: string): string {
  const pieces: string[] = [];
  let hidden = 0;
  const parser = new Parser({
    onopentag(name) {
      if (hiddenTags.has(name)) hidden++;
      if (hidden === 0 && blockTags.has(name)) pieces.push('\n\n');
    },
    onclosetag(name) {
      if (hiddenTags.has(name)) hidden--;
      if (hidden === 0 && blockTags.has(name)) pieces.push('\n\n');
    },
    ontext(text) {
      if (hidden === 0) pieces.push(text.replace(/\s+/g, ' '));
    },
  });
  parser.end(html.slice(0, maxArticleLength * 2));
  return pieces.join('').replace(/ +/g, ' ').replace(/ *\n */g, '\n').replace(/\n{3,}/g, '\n\n').trim();
}

const text = (value: unknown) => typeof value === 'string' ? value : '';
const label = (value: unknown) => htmlText(text(value)).replace(/\s+/g, ' ').slice(0, 240);
const object = (value: unknown): value is Record<string, unknown> =>
  !!value && typeof value === 'object' && !Array.isArray(value);

function storyDate(value: unknown, planet = false): number | null {
  const time = planet && typeof value === 'number' ? (value + 978307200) * 1000 : Date.parse(text(value));
  return Number.isFinite(new Date(time).getTime()) ? time : null;
}

function story(input: Omit<FeedEntry, 'id'> & { id?: string }): FeedEntry | null {
  if (!input.title && !input.content && !input.url) return null;
  const id = input.id || input.url || JSON.stringify([input.title, input.publishedAt, input.content.slice(0, 120)]);
  return { ...input, id: id.slice(0, 2048), content: input.content.slice(0, maxArticleLength) };
}

function compact(entries: (FeedEntry | null)[]): FeedEntry[] {
  const unique = new Map<string, FeedEntry>();
  for (const entry of entries) {
    if (entry && !unique.has(entry.id)) unique.set(entry.id, entry);
  }
  return Array.from(unique.values());
}

function parseJSON(document: string, url: string): Feed | null {
  let data: unknown;
  try {
    data = JSON.parse(document);
  } catch {
    return null;
  }
  if (!object(data)) return null;
  if (text(data.version).startsWith('https://jsonfeed.org/version/') && Array.isArray(data.items)) {
    return {
      title: label(data.title),
      entries: compact(data.items.slice(0, maxEntries).filter(object).map((entry) => story({
        id: text(entry.id),
        title: label(entry.title),
        content: text(entry.content_text).trim() || htmlText(text(entry.content_html)) || text(entry.summary),
        format: 'text',
        url: httpsURL(text(entry.url) || text(entry.external_url), url)?.href ?? '',
        publishedAt: storyDate(entry.date_published ?? entry.date_modified),
      }))),
    };
  }
  if (!text(data.name).trim() || !Array.isArray(data.articles)) return null;
  return {
    title: label(data.name),
    entries: compact(data.articles.slice(0, maxEntries).filter(object).map((entry) => {
      const id = text(entry.id);
      const relative = text(entry.link) || (id ? encodeURIComponent(id) + '/' : '');
      const markdown = text(entry.content);
      const link = httpsURL(text(entry.externalLink), url) ?? httpsURL(relative.replace(/^\/(?!\/)/, ''), url);
      return story({
        id,
        title: label(entry.title),
        content: markdown || htmlText(text(entry.contentRendered)),
        format: markdown ? 'markdown' : 'text',
        url: link?.href ?? '',
        publishedAt: storyDate(entry.created ?? entry.modified, true),
      });
    })),
  };
}

function elements(parent: { children: Node[] }, name?: string): Element[] {
  return parent.children.filter((node): node is Element => 'attribs' in node)
    .filter((node) => name === undefined || node.name.toLowerCase().split(':').pop() === name);
}

function field(parent: Element, name: string) {
  const node = elements(parent, name)[0];
  return node ? DomUtils.textContent(node).trim() : '';
}

function entryBody(entry: Element): string {
  const content = elements(entry).find((tag) => tag.name.toLowerCase() === 'content:encoded')
    ?? elements(entry, 'content')[0] ?? elements(entry, 'description')[0] ?? elements(entry, 'summary')[0];
  if (!content) return '';
  const raw = elements(content).length ? DomUtils.getInnerHTML(content) : DomUtils.textContent(content);
  return content.attribs.type === 'text' ? raw.trim() : htmlText(raw);
}

function parseXML(document: string, url: string): Feed | null {
  const root = elements(parseDocument(document, { xmlMode: true }))[0];
  if (!root) return null;
  const kind = root.name.toLowerCase().split(':').pop();
  if (kind !== 'rss' && kind !== 'rdf' && kind !== 'feed') return null;
  const atom = kind === 'feed';
  const channel = atom ? root : elements(root, 'channel')[0];
  if (!channel) return null;
  const sourceBase = httpsURL(root.attribs['xml:base'] ?? '', url)?.href ?? url;
  const base = httpsURL(channel.attribs['xml:base'] ?? '', sourceBase)?.href ?? sourceBase;
  return {
    title: label(field(channel, 'title')),
    entries: compact(elements(kind === 'rdf' ? root : channel, atom ? 'entry' : 'item').slice(0, maxEntries).map((entry) => {
      const entryBase = httpsURL(entry.attribs['xml:base'] ?? '', base)?.href ?? base;
      const link = atom ? elements(entry, 'link').find((node) => !node.attribs.rel || node.attribs.rel === 'alternate') : undefined;
      const linkBase = httpsURL(link?.attribs['xml:base'] ?? '', entryBase)?.href ?? entryBase;
      return story({
        id: field(entry, atom ? 'id' : 'guid'),
        title: label(field(entry, 'title')),
        content: entryBody(entry),
        format: 'text',
        url: httpsURL(link?.attribs.href ?? field(entry, 'link'), linkBase)?.href ?? '',
        publishedAt: storyDate(field(entry, atom ? 'published' : 'pubdate') || field(entry, atom ? 'updated' : 'date')),
      });
    })),
  };
}

export function parseFeed(document: string, url: string): Feed | null {
  const body = document.trim();
  const result = body.startsWith('{') ? parseJSON(body, url) : body.startsWith('<') ? parseXML(body, url) : null;
  return result && { ...result, title: result.title || new URL(url).hostname };
}

export function discoverFeedURLs(html: string, url: string): string[] {
  const types = new Set(['application/rss+xml', 'application/atom+xml', 'application/feed+json', 'application/json']);
  const dom = parseDocument(html);
  const baseTag = DomUtils.findOne((node) => node.name === 'base', dom.children);
  const base = httpsURL(baseTag?.attribs.href ?? '', url)?.href ?? url;
  const links = DomUtils.findAll((node) => node.name === 'link'
    && (node.attribs.rel ?? '').toLowerCase().split(/\s+/).includes('alternate')
    && types.has((node.attribs.type ?? '').toLowerCase().split(';')[0]), dom.children);
  return Array.from(new Set(links.flatMap((link) => {
    const target = httpsURL(link.attribs.href ?? '', base);
    return target ? [target.href] : [];
  })));
}
