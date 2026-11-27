import { useFocusEffect, useNavigation } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { useSQLiteContext } from 'expo-sqlite';
import { ArrowUpRight, FileText, PenLine, Plus } from 'lucide-react-native';
import { useCallback, useState } from 'react';
import { Alert, FlatList, Pressable, StyleSheet, Text, View } from 'react-native';
import { copy, errorMessage, formatDate } from './copy';
import type { Routes } from './routes';
import { createDraft, type DraftSummary, listDrafts } from './storage';
import { serif, usePalette } from './theme';
import { Button, EmptyState, layout, Notice, Screen, ScreenHeading, Segments } from './ui';

export function WritingScreen() {
  const db = useSQLiteContext();
  const navigation = useNavigation<NativeStackNavigationProp<Routes>>();
  const { colors } = usePalette();
  const [items, setItems] = useState<DraftSummary[]>([]);
  const [tab, setTab] = useState<'drafts' | 'published'>('drafts');
  const [error, setError] = useState('');
  useFocusEffect(useCallback(() => {
    try {
      setItems(listDrafts(db));
      setError('');
    } catch (cause) {
      setError(errorMessage(cause));
    }
  }, [db]));

  function addDraft() {
    try {
      const id = createDraft(db);
      setTab('drafts');
      navigation.navigate('Draft', { id });
    } catch (cause) {
      Alert.alert(errorMessage(cause));
    }
  }

  return <Screen tab>
    <FlatList data={items.filter((item) => (item.cid !== null) === (tab === 'published'))}
      keyExtractor={(item) => item.id} contentContainerStyle={layout.page}
      ListHeaderComponent={
        <>
          <ScreenHeading title={copy.desk} note={formatDate(Date.now(), true)}
            actions={<Button label={copy.newDraft} icon={Plus} onPress={addDraft} />} />
          <Segments tabs={[{ id: 'drafts', label: copy.drafts }, { id: 'published', label: copy.published }]}
            value={tab} onChange={setTab} />
          {error && <Notice text={error} error />}
        </>
      }
      ListEmptyComponent={tab === 'drafts'
        ? <EmptyState icon={PenLine} title={copy.noDrafts} body={copy.noDraftsBody} />
        : <EmptyState icon={FileText} title={copy.noPublished} body={copy.noPublishedBody} />}
      renderItem={({ item }) =>
        <Pressable accessibilityRole="button" onPress={() => navigation.navigate('Draft', { id: item.id })}
          style={({ pressed }) => [styles.card, { backgroundColor: pressed ? colors.soft : colors.paper, borderColor: colors.line }]}>
          <View style={layout.flex}>
            <Text style={[layout.small, { color: colors.accent }]} numberOfLines={1}>
              {item.cid ? copy.published : item.siteId ? copy.pending : copy.draft} · {formatDate(item.updatedAt)}
            </Text>
            <Text style={[styles.title, { color: colors.text }]} numberOfLines={2}>{item.title.trim() || copy.untitled}</Text>
            <Text style={[layout.small, { color: colors.muted }]} numberOfLines={2}>
              {item.preview.trim().replace(/\s+/g, ' ') || copy.emptyDraft}
            </Text>
          </View>
          <ArrowUpRight size={20} color={colors.accent} accessible={false} />
        </Pressable>
      } />
  </Screen>;
}

const styles = StyleSheet.create({
  card: {
    marginTop: 12, flexDirection: 'row', alignItems: 'center', gap: 12,
    borderWidth: 1, borderRadius: 11, padding: 17,
  },
  title: { fontFamily: serif, fontSize: 23, fontWeight: '600', marginVertical: 4 },
});
