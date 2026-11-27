import assert from 'node:assert/strict';
import { test } from 'node:test';
import { parseFeed } from './feed-format.ts';
import { loadFeed, subscriptionURL } from './feeds.ts';

test('RSS caches article text and keeps guid stable across refreshes', () => {
  const feed = parseFeed(`<?xml version="1.0"?>
    <rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/"><channel>
      <title>A &amp; B</title>
      <item><guid>story-1</guid><title>First</title><link>/first</link>
        <pubDate>Fri, 27 Nov 2026 10:00:00 GMT</pubDate>
        <content:encoded><![CDATA[<p>One</p><script>hidden()</script><p>Two &amp; three</p>]]></content:encoded>
      </item>
      <item><guid>story-1</guid><title>Duplicate</title></item>
    </channel></rss>`, 'https://example.org/feed.xml');
  assert.deepEqual(feed, {
    title: 'A & B',
    entries: [{
      id: 'story-1', title: 'First', url: 'https://example.org/first', content: 'One\n\nTwo & three',
      format: 'text', publishedAt: Date.UTC(2026, 10, 27, 10),
    }],
  });
});

test('Atom links honor xml:base and prefer the original article', () => {
  const feed = parseFeed(`<feed xmlns="http://www.w3.org/2005/Atom" xml:base="https://example.org/writing/">
    <title>Journal</title><entry><id>urn:example:one</id><title>One</title>
    <link rel="enclosure" href="audio.mp3" /><link rel="alternate" href="first.html" />
    <content type="text">a &lt; b</content></entry></feed>`, 'https://feeds.example.org/atom.xml');
  assert.equal(feed?.entries[0]?.url, 'https://example.org/writing/first.html');
  assert.equal(feed?.entries[0]?.content, 'a < b');
});

test('Planet metadata retains Markdown, the Apple epoch and the IPNS path', () => {
  const url = subscriptionURL('planet://k51example');
  assert.equal(url, 'https://eth.sucks/ipns/k51example/planet.json');
  assert.equal(subscriptionURL('blog.eth'), 'https://blog.eth.limo/');
  const feed = parseFeed(JSON.stringify({
    name: 'Journal', articles: [{ id: 'abc', link: '/hello/', title: 'Hello', content: '*World*', created: 0 }],
  }), url);
  assert.deepEqual(feed?.entries[0], {
    id: 'abc', title: 'Hello', content: '*World*', format: 'markdown', publishedAt: Date.UTC(2001, 0, 1),
    url: 'https://eth.sucks/ipns/k51example/hello/',
  });
});

test('homepage discovery follows HTTPS redirects and rejects unsafe article URLs', async () => {
  const requests: string[] = [];
  const responses = new Map([
    ['https://example.org/', new Response(null, { status: 308, headers: { Location: '/blog/' } })],
    ['https://example.org/blog/', new Response('<html><head><link rel="alternate" type="application/feed+json" href="feed.json"></head></html>')],
    ['https://example.org/blog/feed.json', Response.json({
      version: 'https://jsonfeed.org/version/1.1', title: 'Blog',
      items: [{ id: '1', title: 'First', url: 'javascript:alert(1)', content_html: '<p>Text</p>' }],
    })],
  ]);
  const result = await loadFeed('https://example.org', async (url, options) => {
    assert.equal(options?.redirect, 'manual');
    assert.equal(options?.credentials, 'omit');
    requests.push(String(url));
    return responses.get(String(url)) ?? new Response(null, { status: 404 });
  });
  assert.deepEqual(requests, [...responses.keys()]);
  assert.equal(result.url, 'https://example.org/blog/feed.json');
  assert.equal(result.entries[0]?.url, '');
  assert.equal(result.entries[0]?.content, 'Text');
});

test('feed loading stops at a cleartext redirect and caps response sizes', async () => {
  assert.throws(() => subscriptionURL('http://example.org/rss'), { message: 'invalid_feed_url' });
  let calls = 0;
  await assert.rejects(loadFeed('https://example.org/rss', async () => {
    calls++;
    return new Response(null, { status: 302, headers: { Location: 'http://example.org/feed' } });
  }), { message: 'feed_http' });
  assert.equal(calls, 1);
  await assert.rejects(loadFeed('https://example.org/rss', async () =>
    new Response('', { headers: { 'content-length': String(5 * 1024 * 1024) } })),
    { message: 'response_too_large' });
});
