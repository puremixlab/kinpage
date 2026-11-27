import { useFocusEffect, useNavigation } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { useSQLiteContext } from 'expo-sqlite';
import { Bookmark, BookOpen, List, Plus, RefreshCw } from 'lucide-react-native';
import { useCallback, useRef, useState } from 'react';
import { FlatList, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { copy, errorMessage, formatDate } from './copy';
import { loadFeed } from './feeds';
import type { Routes } from './routes';
import { type EntrySummary, listEntries, listSubscriptions, type ReadingFilter, saveFeed, type Subscription } from './storage';
import { serif, usePalette } from './theme';
import { Button, EmptyState, IconButton, layout, Notice, Screen, ScreenHeading, Segments } from './ui';

export function ReadingScreen() {
  const db = useSQLiteContext();
  const navigation = useNavigation<NativeStackNavigationProp<Routes>>();
  const { colors } = usePalette();
  const [sources, setSources] = useState<Subscription[]>([]);
  const [entries, setEntries] = useState<EntrySummary[]>([]);
  const [filter, setFilter] = useState<ReadingFilter>('all');
  const [source, setSource] = useState<string | null>(null);
  const [refreshing, setRefreshing] = useState(false);
  const activeRefresh = useRef(false);
  const [error, setError] = useState('');

  const readCache = useCallback((view: ReadingFilter, sourceId: string | null) => {
    try {
      const existing = listSubscriptions(db);
      setSources(existing);
      const selection = existing.some((item) => item.id === sourceId) ? sourceId : null;
      if (selection !== sourceId) setSource(selection);
      setEntries(listEntries(db, view, selection));
    } catch (cause) {
      setError(errorMessage(cause));
    }
  }, [db]);
  useFocusEffect(useCallback(() => readCache(filter, source), [filter, source, readCache]));
  const selection = useRef({ filter, source });
  selection.current = { filter, source };

  async function refresh() {
    if (activeRefresh.current || sources.length === 0) return;
    activeRefresh.current = true;
    setRefreshing(true);
    setError('');
    const targets = source ? sources.filter((item) => item.id === source) : sources;
    let failed = 0;
    try {
      for (let offset = 0; offset < targets.length; offset += 3) {
        const results = await Promise.allSettled(targets.slice(offset, offset + 3).map(async (item) => {
          saveFeed(db, await loadFeed(item.url, fetch, false), item.id);
        }));
        failed += results.filter((result) => result.status === 'rejected').length;
      }
      if (failed) setError(copy.refreshFailed(failed));
      readCache(selection.current.filter, selection.current.source);
    } finally {
      activeRefresh.current = false;
      setRefreshing(false);
    }
  }

  return <Screen tab>
    <FlatList data={entries} keyExtractor={(item) => JSON.stringify([item.subscriptionId, item.entryId])}
      contentContainerStyle={layout.page} refreshing={refreshing} onRefresh={refresh}
      ListHeaderComponent={<>
        <ScreenHeading title={copy.reading} actions={<>
          <Button label={copy.addFeed} icon={Plus} onPress={() => navigation.navigate('Subscribe')} />
          {sources.length > 0 && <>
            <IconButton label={copy.refresh} icon={RefreshCw} disabled={refreshing} onPress={refresh} />
            <IconButton label={copy.manageFeeds} icon={List} onPress={() => navigation.navigate('Subscriptions')} />
          </>}
        </>} />
        <Segments tabs={[
          { id: 'all', label: copy.all }, { id: 'unread', label: copy.unread }, { id: 'starred', label: copy.starred },
        ]} value={filter} onChange={setFilter} />
        {sources.length > 0 && <ScrollView horizontal showsHorizontalScrollIndicator={false}
          contentContainerStyle={styles.sourceList}>
          {[{ id: null, title: copy.subscriptions }, ...sources].map(({ id, title }) =>
            <Pressable accessibilityRole="button" accessibilityState={{ selected: id === source }}
              key={id ?? 'all-sources'} onPress={() => setSource(id)}
              style={[styles.chip, { borderColor: colors.line, backgroundColor: id === source ? colors.soft : colors.paper }]}>
              <Text numberOfLines={1} style={[layout.small, { color: id === source ? colors.text : colors.muted }]}>{title}</Text>
            </Pressable>
          )}
        </ScrollView>}
        {error && <Notice text={error} error />}
      </>}
      ListEmptyComponent={<EmptyState icon={BookOpen} title={sources.length ? copy.noEntries : copy.noFeeds}
        body={sources.length ? copy.noEntriesBody : copy.noFeedsBody} />}
      renderItem={({ item }) =>
        <Pressable accessibilityRole="button" accessibilityHint={item.readAt === null ? copy.unread : undefined}
          style={({ pressed }) => [styles.row, { borderBottomColor: colors.line, backgroundColor: pressed ? colors.soft : 'transparent' }]}
          onPress={() => navigation.navigate('Article', { entryId: item.entryId, subscriptionId: item.subscriptionId })}>
          <View style={[styles.dot, { backgroundColor: item.readAt === null ? colors.accent : 'transparent' }]} />
          <View style={layout.flex}>
            <Text numberOfLines={1} style={[layout.small, { color: colors.accent }]}>
              {item.sourceTitle}{item.publishedAt === null ? '' : '  ·  ' + formatDate(item.publishedAt)}
            </Text>
            <Text numberOfLines={2} style={[styles.entryTitle, { color: colors.text }]}>{item.title || copy.untitledEntry}</Text>
            {!!item.preview && <Text numberOfLines={2} style={[layout.small, { color: colors.muted }]}>
              {item.preview.replace(/\s+/g, ' ')}
            </Text>}
          </View>
          {!!item.starred && <Bookmark accessibilityLabel={copy.starred} size={18} color={colors.accent} />}
        </Pressable>
      } />
  </Screen>;
}

const styles = StyleSheet.create({
  sourceList: { gap: 8, paddingVertical: 12 },
  chip: { borderWidth: 1, borderRadius: 18, paddingHorizontal: 13, justifyContent: 'center', minHeight: 44, maxWidth: 190 },
  row: { flexDirection: 'row', alignItems: 'flex-start', gap: 9, paddingVertical: 16, borderBottomWidth: StyleSheet.hairlineWidth },
  dot: { width: 6, height: 6, borderRadius: 3, marginTop: 7 },
  entryTitle: { fontFamily: serif, fontSize: 21, fontWeight: '600', marginVertical: 3 },
});
