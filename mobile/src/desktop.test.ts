import assert from 'node:assert/strict';
import { test } from 'node:test';
import { listDesktopSites, normalizeConnection, publishArticle, publishedURL } from './desktop.ts';

const siteId = '293ad59a-15a2-4601-be81-efded07451a2';
const articleId = 'ef62d160-8da1-41fb-b397-8b19c246086d';
const cid = 'bafybeihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku';
const connection = normalizeConnection('https://journal.example.org/', 'Planet', 'local-test-passcode');

test('desktop credentials can only be sent to an explicit HTTPS origin', () => {
  for (const url of ['http://journal.example.org', 'https://user@journal.example.org', 'https://example.org/forward', 'https://example.org/?q=1']) {
    assert.throws(() => normalizeConnection(url, 'Planet', 'test'), { message: 'invalid_desktop_url' });
  }
  assert.throws(() => normalizeConnection(connection.origin, 'Planet:test', 'test'), { message: 'invalid_credentials' });
});

test('site lookup authenticates without following redirects or saving cookies', async () => {
  assert.deepEqual(await listDesktopSites(connection, async (url, init) => {
    assert.equal(url, connection.origin + '/v0/mobile/sites');
    assert.equal(init?.redirect, 'error');
    assert.equal(init?.credentials, 'omit');
    assert.equal(new Headers(init?.headers).get('authorization'), 'Basic ' + btoa('Planet:local-test-passcode'));
    return Response.json([{ id: siteId, name: 'Journal', hidden: 'omitted' }]);
  }), [{ id: siteId, name: 'Journal' }]);
  await assert.rejects(listDesktopSites(connection, async () => new Response(null, { status: 403 })),
    { message: 'auth_disabled' });
});

test('publication reuses the same PUT and verifies the returned article ID', async () => {
  const article = { id: articleId, title: ' 手机草稿 ', content: 'Article body\n' };
  const requests: (RequestInit | undefined)[] = [];
  const send: typeof fetch = async (url, init) => {
    assert.equal(url, connection.origin + '/v0/mobile/sites/' + siteId + '/articles/' + articleId);
    requests.push(init);
    return requests.length === 1 ? new Response(null, { status: 502 })
      : Response.json({ articleId: articleId.toUpperCase(), cid });
  };
  await assert.rejects(publishArticle(connection, siteId, article, send), { message: 'publish_failed' });
  const receipt = await publishArticle(connection, siteId, article, send);
  assert.deepEqual(receipt, { articleId: articleId.toUpperCase(), cid });
  assert.equal(requests[0]?.method, 'PUT');
  assert.equal(requests[0]?.body, requests[1]?.body);
  assert.deepEqual(JSON.parse(String(requests[1]?.body)), { title: '手机草稿', content: 'Article body\n' });
  assert.equal(publishedURL(cid, articleId), 'https://eth.sucks/ipfs/' + cid + '/' + receipt.articleId + '/');
  await assert.rejects(publishArticle(connection, siteId, article, async () => Response.json({ articleId: siteId, cid })),
    { message: 'desktop_error' });
});
