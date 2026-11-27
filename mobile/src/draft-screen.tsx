import { useFocusEffect, usePreventRemove } from '@react-navigation/native';
import { useHeaderHeight } from '@react-navigation/elements';
import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { useSQLiteContext } from 'expo-sqlite';
import { ArrowUpRight, CircleCheck, Copy, Share2, Trash2, Upload } from 'lucide-react-native';
import { useCallback, useState } from 'react';
import { Alert, Keyboard, KeyboardAvoidingView, Platform, Share, StyleSheet, Text, TextInput, View } from 'react-native';
import { copy, errorMessage } from './copy';
import { publishedURL } from './desktop';
import { MarkdownBody, openLink } from './markdown';
import type { Routes } from './routes';
import { createDraft, deleteDraft, type Draft, editDraft, getDraft } from './storage';
import { serif, usePalette } from './theme';
import { Button, IconButton, layout, Notice, Page, Screen, Segments } from './ui';

export function DraftScreen({ navigation, route }: NativeStackScreenProps<Routes, 'Draft'>) {
  const db = useSQLiteContext();
  const headerHeight = useHeaderHeight();
  const { colors } = usePalette();
  const [draft, setDraft] = useState<Draft | null>(() => getDraft(db, route.params.id));
  const [mode, setMode] = useState<'edit' | 'preview'>('edit');
  const [saveFailed, setSaveFailed] = useState(false);
  useFocusEffect(useCallback(() => {
    setDraft(getDraft(db, route.params.id));
  }, [db, route.params.id]));
  usePreventRemove(saveFailed, ({ data }) => {
    Alert.alert(copy.saveFailed, undefined, [
      { text: copy.cancel, style: 'cancel' },
      { text: copy.retry, onPress: () => draft && save(draft) },
      { text: copy.back, style: 'destructive', onPress: () => navigation.dispatch(data.action) },
    ]);
  });

  function save(changed: Draft) {
    try {
      const updatedAt = editDraft(db, changed.id, changed.title, changed.content);
      setDraft({ ...changed, updatedAt });
      setSaveFailed(false);
    } catch {
      setDraft(changed);
      setSaveFailed(true);
    }
  }

  if (!draft) return <Screen><Page><Notice text={copy.noArticle} error /></Page></Screen>;
  const current = draft;
  const editable = !current.siteId;
  const link = current.cid ? publishedURL(current.cid, current.id) : '';

  function shareDraft() {
    const message = (current.title.trim() ? '# ' + current.title.trim() + '\n\n' : '') + current.content;
    if (message.trim()) void Share.share({ message }).catch((error) => Alert.alert(errorMessage(error)));
  }

  function removeLocalCopy() {
    Alert.alert(copy.removeDraft, copy.removeDraftBody, [
      { text: copy.cancel, style: 'cancel' },
      { text: copy.remove, style: 'destructive', onPress: () => {
        try {
          deleteDraft(db, current.id);
          navigation.goBack();
        } catch (error) {
          Alert.alert(errorMessage(error));
        }
      } },
    ]);
  }

  function duplicate() {
    try {
      const id = createDraft(db, current.title, current.content);
      navigation.push('Draft', { id });
    } catch (error) {
      Alert.alert(errorMessage(error));
    }
  }

  return <Screen>
    <KeyboardAvoidingView style={layout.flex} behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      keyboardVerticalOffset={headerHeight}>
      <Page>
        <View style={layout.wrap}>
          {editable && <View style={layout.flex}>
            <Segments tabs={[{ id: 'edit', label: copy.edit }, { id: 'preview', label: copy.preview }]}
              value={mode} onChange={(value) => { Keyboard.dismiss(); setMode(value); }} />
          </View>}
          <IconButton label={copy.shareText} icon={Share2} onPress={shareDraft}
            disabled={!current.title.trim() && !current.content.trim()} />
          <IconButton label={copy.removeDraft} icon={Trash2} onPress={removeLocalCopy} disabled={saveFailed} />
        </View>

        {editable && mode === 'edit' ?
          <View style={[styles.paper, { backgroundColor: colors.paper, borderColor: colors.line }]}>
            <TextInput accessibilityLabel={copy.title} placeholder={copy.title} placeholderTextColor={colors.muted}
              value={current.title} onChangeText={(title) => save({ ...current, title })} maxLength={160}
              style={[styles.titleInput, { color: colors.text, borderBottomColor: colors.line }]} />
            <TextInput accessibilityLabel={copy.body} placeholder={copy.body} placeholderTextColor={colors.muted}
              multiline textAlignVertical="top" scrollEnabled={false} maxLength={60_000}
              value={current.content} onChangeText={(content) => save({ ...current, content })}
              style={[styles.bodyInput, { color: colors.text }]} />
          </View>
          : <View style={[styles.paper, { backgroundColor: colors.paper, borderColor: colors.line }]}>
            <Text accessibilityRole="header" style={[layout.headline, { color: colors.text, marginBottom: 15 }]}>
              {current.title.trim() || copy.untitled}
            </Text>
            {current.content.trim() ? <MarkdownBody value={current.content} />
              : <Text style={[layout.body, { color: colors.muted }]}>{copy.emptyDraft}</Text>}
          </View>}

        {saveFailed ? <>
          <Notice text={copy.saveFailed} error />
          <Button label={copy.retry} tone="quiet" onPress={() => save(current)} />
        </> : editable && <View style={layout.row}>
          <CircleCheck size={16} color={colors.accent} accessible={false} />
          <Text style={[layout.small, { color: colors.muted }]}>{copy.saved}</Text>
        </View>}
        {!!current.siteName && <Text style={[layout.small, { color: colors.muted }]}>
          {link ? copy.published : copy.pending} · {current.siteName}
        </Text>}
        <Button label={link ? copy.openArticle : editable ? copy.publish : copy.continuePublish}
          icon={link ? ArrowUpRight : Upload} disabled={!link && (saveFailed || !current.title.trim() || !current.content.trim())}
          onPress={() => {
            Keyboard.dismiss();
            if (link) openLink(link);
            else navigation.navigate('Publish', { id: current.id });
          }} />
        {!editable && <Button label={copy.duplicateDraft} icon={Copy} tone="quiet" onPress={duplicate} />}
      </Page>
    </KeyboardAvoidingView>
  </Screen>;
}

const styles = StyleSheet.create({
  paper: { borderRadius: 12, borderWidth: 1, padding: 15 },
  titleInput: { fontFamily: serif, fontWeight: '600', fontSize: 27, minHeight: 55, borderBottomWidth: StyleSheet.hairlineWidth },
  bodyInput: { fontSize: 17, lineHeight: 26, paddingTop: 12, minHeight: 235 },
});
