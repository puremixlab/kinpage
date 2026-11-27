import { useFocusEffect, usePreventRemove } from '@react-navigation/native';
import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { useSQLiteContext } from 'expo-sqlite';
import { ArrowUpRight, Bookmark, BookOpen, Check, Plus, Trash2 } from 'lucide-react-native';
import { useCallback, useEffect, useRef, useState } from 'react';
import { Alert, Keyboard, StyleSheet, Text, View } from 'react-native';
import { copy, errorMessage, formatDate } from './copy';
import { loadFeed } from './feeds';
import { MarkdownBody, openLink } from './markdown';
import type { Routes } from './routes';
import {
  deleteSubscription, getEntry, listSubscriptions, markRead, saveFeed, setStarred, type Subscription,
} from './storage';
import { serif, usePalette } from './theme';
import { Button, EmptyState, Field, IconButton, layout, Notice, Page, Screen } from './ui';

export function SubscribeScreen({ navigation }: NativeStackScreenProps<Routes, 'Subscribe'>) {
  const db = useSQLiteContext();
  const { colors } = usePalette();
  const [url, setURL] = useState('');
  const [working, setWorking] = useState(false);
  const [error, setError] = useState('');
  const active = useRef(false);
  usePreventRemove(working, () => {});

  async function subscribe() {
    if (active.current) return;
    active.current = true;
    setError('');
    setWorking(true);
    Keyboard.dismiss();
    let saved = false;
    try {
      saveFeed(db, await loadFeed(url));
      saved = true;
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      active.current = false;
      setWorking(false);
    }
    if (saved) setDone(true);
  }

  const [done, setDone] = useState(false);
  useEffect(() => {
    if (done && !working) navigation.goBack();
  }, [done, navigation, working]);

  return <Screen><Page>
    <Field label={copy.feedURL} keyboardType="url" autoCapitalize="none" autoCorrect={false}
      placeholder="https://example.org/feed.xml" maxLength={2048} value={url} onChangeText={setURL} editable={!working} />
    <Text style={[layout.body, { color: colors.muted }]}>{copy.feedHint}</Text>
    {error && <Notice text={error} error />}
    <Button label={working ? copy.following : copy.subscribe} icon={Plus} onPress={subscribe} busy={working} disabled={!url.trim()} />
  </Page></Screen>;
}

export function SubscriptionsScreen({ navigation }: NativeStackScreenProps<Routes, 'Subscriptions'>) {
  const db = useSQLiteContext();
  const { colors } = usePalette();
  const [sources, setSources] = useState<Subscription[]>([]);
  const reload = useCallback(() => setSources(listSubscriptions(db)), [db]);
  useFocusEffect(reload);

  function remove(item: Subscription) {
    Alert.alert(copy.removeFeed, copy.removeFeedBody, [
      { text: copy.cancel, style: 'cancel' },
      { text: copy.remove, style: 'destructive', onPress: () => {
        try {
          deleteSubscription(db, item.id);
          reload();
        } catch (error) {
          Alert.alert(errorMessage(error));
        }
      } },
    ]);
  }

  return <Screen><Page>
    <Button label={copy.addFeed} icon={Plus} onPress={() => navigation.navigate('Subscribe')} />
    {sources.map((item) =>
      <View key={item.id} style={[layout.row, styles.source, { backgroundColor: colors.paper, borderColor: colors.line }]}>
        <View style={layout.flex}>
          <Text style={[layout.label, { color: colors.text }]}>{item.title}</Text>
          <Text selectable style={[layout.small, { color: colors.muted }]} numberOfLines={2}>{item.url}</Text>
        </View>
        <IconButton label={copy.remove + ' ' + item.title} icon={Trash2} onPress={() => remove(item)} />
      </View>
    )}
    {!sources.length && <EmptyState icon={BookOpen} title={copy.noFeeds} body={copy.noFeedsBody} />}
  </Page></Screen>;
}

export function ArticleScreen({ route }: NativeStackScreenProps<Routes, 'Article'>) {
  const db = useSQLiteContext();
  const { colors } = usePalette();
  const key = route.params;
  const [entry, setEntry] = useState(() => getEntry(db, key));
  useEffect(() => {
    try {
      markRead(db, key, true);
      setEntry(getEntry(db, key));
    } catch (error) {
      Alert.alert(errorMessage(error));
    }
  }, [db, key]);
  if (!entry) return <Screen><Page><Notice text={copy.noArticle} /></Page></Screen>;

  function update(operation: () => void) {
    try {
      operation();
      setEntry(getEntry(db, key));
    } catch (error) {
      Alert.alert(errorMessage(error));
    }
  }

  return <Screen><Page>
    <View style={[styles.articleHead, { borderBottomColor: colors.line }]}>
      <Text style={[layout.label, { color: colors.accent }]}>{entry.sourceTitle}</Text>
      <Text accessibilityRole="header" style={[styles.title, { color: colors.text }]}>{entry.title || copy.untitledEntry}</Text>
      {entry.publishedAt !== null && <Text style={[layout.small, { color: colors.muted }]}>{formatDate(entry.publishedAt)}</Text>}
      <View style={[layout.wrap, { marginTop: 8 }]}>
        <Button label={entry.starred ? copy.unsaveArticle : copy.saveArticle} icon={Bookmark} tone="quiet"
          onPress={() => update(() => setStarred(db, key, !entry.starred))} />
        <Button label={entry.readAt === null ? copy.markRead : copy.markUnread} icon={Check} tone="quiet"
          onPress={() => update(() => markRead(db, key, entry.readAt === null))} />
      </View>
    </View>
    {entry.format === 'markdown' && entry.content.trim()
      ? <MarkdownBody value={entry.content} baseUrl={entry.url || undefined} />
      : <Text selectable style={[styles.articleBody, { color: colors.text }]}>{entry.content || copy.noContent}</Text>}
    {!!entry.url && <Button label={copy.original} icon={ArrowUpRight} onPress={() => openLink(entry.url)} />}
  </Page></Screen>;
}

const styles = StyleSheet.create({
  source: { padding: 13, borderWidth: 1, borderRadius: 10 },
  articleHead: { gap: 8, paddingBottom: 16, borderBottomWidth: StyleSheet.hairlineWidth },
  title: { fontFamily: serif, fontSize: 34, fontWeight: '600' },
  articleBody: { fontSize: 18, lineHeight: 29 },
});
