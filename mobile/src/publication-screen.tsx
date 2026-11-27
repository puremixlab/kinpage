import { useFocusEffect, usePreventRemove } from '@react-navigation/native';
import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { useSQLiteContext } from 'expo-sqlite';
import { ArrowUpRight, Check, Monitor, Share2, Upload } from 'lucide-react-native';
import { useCallback, useRef, useState } from 'react';
import { Alert, Pressable, Share, StyleSheet, Text, View } from 'react-native';
import { getConnection } from './connection-storage';
import { copy, errorMessage } from './copy';
import { type Connection, type DesktopSite, listDesktopSites, publishArticle, publishedURL } from './desktop';
import { openLink } from './markdown';
import type { Routes } from './routes';
import { beginPublication, type Draft, endPublication, getDraft } from './storage';
import { usePalette } from './theme';
import { Button, EmptyState, layout, Loading, Notice, Page, Screen } from './ui';

export function PublicationScreen({ navigation, route }: NativeStackScreenProps<Routes, 'Publish'>) {
  const db = useSQLiteContext();
  const { colors } = usePalette();
  const id = route.params.id;
  const [draft, setDraft] = useState<Draft | null>(() => getDraft(db, id));
  const [connection, setConnection] = useState<Connection | null>(null);
  const [sites, setSites] = useState<DesktopSite[]>([]);
  const [selected, setSelected] = useState<string>();
  const [loading, setLoading] = useState(true);
  const [working, setWorking] = useState(false);
  const [error, setError] = useState('');
  const [siteRequest, setSiteRequest] = useState(0);
  const active = useRef(false);
  usePreventRemove(working, () => {});

  useFocusEffect(useCallback(() => {
    const current = getDraft(db, id);
    let visible = true;
    setDraft(current);
    setError('');
    setSites([]);
    setLoading(!current?.cid);
    if (current?.cid) return;
    void (async () => {
      try {
        const conn = await getConnection();
        if (!visible) return;
        setConnection(conn);
        if (conn && !current?.siteId) {
          const available = await listDesktopSites(conn);
          if (!visible) return;
          setSites(available);
          setSelected((existing) => available.find(({ id }) => id === existing)?.id ?? available[0]?.id);
        }
      } catch (cause) {
        if (visible) setError(errorMessage(cause));
      } finally {
        if (visible) setLoading(false);
      }
    })();
    return () => { visible = false; };
  }, [db, id, siteRequest]));

  async function publish(conn: Connection, target: DesktopSite) {
    if (active.current) return;
    active.current = true;
    setWorking(true);
    setError('');
    try {
      const current = getDraft(db, id);
      if (!current || current.cid) return;
      if (current.siteId) {
        if (current.siteId !== target.id || current.desktopOrigin !== conn.origin) throw new Error('Publication target changed');
      } else if (!beginPublication(db, id, { desktopOrigin: conn.origin, siteId: target.id, siteName: target.name })) {
        throw new Error('Could not save publication target');
      }
      setDraft(getDraft(db, id));
      const receipt = await publishArticle(conn, target.id, current);
      endPublication(db, id, receipt.cid);
      setDraft(getDraft(db, id));
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      active.current = false;
      setWorking(false);
    }
  }

  if (!draft) return <Screen><Page><Notice text={copy.noArticle} error /></Page></Screen>;
  const link = draft.cid ? publishedURL(draft.cid, draft.id) : '';
  if (link) return <Screen><Page>
    <EmptyState icon={Check} title={copy.publicationDone} body={draft.title} />
    <Button label={copy.openArticle} icon={ArrowUpRight} onPress={() => openLink(link)} />
    <Button label={copy.shareLink} icon={Share2} tone="quiet" onPress={() => {
      void Share.share({ message: link }).catch((cause) => Alert.alert(errorMessage(cause)));
    }} />
  </Page></Screen>;

  if (loading) return <Screen><Page><Loading /></Page></Screen>;
  const connectionMatch = connection && (!draft.desktopOrigin || connection.origin === draft.desktopOrigin) ? connection : null;
  const target = draft.siteId && draft.siteName ? { id: draft.siteId, name: draft.siteName } : sites.find((site) => site.id === selected);

  return <Screen><Page>
    <Text accessibilityRole="header" style={[layout.headline, { color: colors.text }]}>{draft.title || copy.untitled}</Text>
    {error && <Notice text={error} error />}
    {!connectionMatch ? <>
      <Notice text={draft.desktopOrigin ? copy.pendingConnection : copy.noConnection} />
      <Button label={copy.configureDesktop} icon={Monitor} onPress={() => navigation.navigate('Desktop')} />
    </> : <>
      {!draft.siteId && sites.length > 0 && <>
        <Text accessibilityRole="header" style={[layout.label, { color: colors.text }]}>{copy.chooseSite}</Text>
        <View style={{ gap: 8 }}>
          {sites.map((site) =>
            <Pressable key={site.id} accessibilityRole="radio" accessibilityState={{ checked: site.id === selected }}
              onPress={() => setSelected(site.id)} disabled={working}
              style={[layout.row, styles.option, { backgroundColor: site.id === selected ? colors.soft : colors.paper, borderColor: colors.line }]}>
              <Text style={[layout.body, layout.flex, { color: colors.text }]}>{site.name}</Text>
              {site.id === selected && <Check size={20} color={colors.accent} accessible={false} />}
            </Pressable>
          )}
        </View>
      </>}
      {draft.siteName && <Notice text={copy.selectedSite + ' · ' + draft.siteName} />}
      {!target && !error && <EmptyState icon={Monitor} title={copy.noSites} body={copy.noSitesBody} />}
      {target && <>
        <Text style={[layout.body, { color: colors.muted }]}>{copy.publishNotice}</Text>
        <Button icon={Upload} label={working ? copy.publishing : draft.siteId ? copy.continuePublish : copy.publish}
          busy={working} onPress={() => {
            if (draft.siteId) void publish(connectionMatch, target);
            else Alert.alert(copy.confirmPublish, copy.publishNotice, [
              { text: copy.cancel, style: 'cancel' },
              { text: copy.publish, onPress: () => { void publish(connectionMatch, target); } },
            ]);
          }} />
      </>}
      {error && <Button label={copy.desktop} icon={Monitor} tone="quiet" disabled={working}
        onPress={() => navigation.navigate('Desktop')} />}
      {error && !target && <Button label={copy.retry} tone="quiet" onPress={() => setSiteRequest((value) => value + 1)} />}
    </>}
  </Page></Screen>;
}

const styles = StyleSheet.create({
  option: { paddingHorizontal: 13, paddingVertical: 8, minHeight: 50, borderWidth: 1, borderRadius: 9 },
});
