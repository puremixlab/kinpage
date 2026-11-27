import type { LucideIcon } from 'lucide-react-native';
import type { PropsWithChildren, ReactNode } from 'react';
import {
  ActivityIndicator, Image, Pressable, ScrollView, StyleSheet, Text, TextInput,
  type TextInputProps, View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { copy } from './copy';
import { serif, usePalette } from './theme';

export const layout = StyleSheet.create({
  flex: { flex: 1 },
  page: { width: '100%', maxWidth: 670, alignSelf: 'center', paddingHorizontal: 22, paddingBottom: 26 },
  stack: { gap: 16 },
  row: { flexDirection: 'row', alignItems: 'center', gap: 10 },
  wrap: { flexDirection: 'row', gap: 8, alignItems: 'center', flexWrap: 'wrap' },
  label: { fontSize: 14, fontWeight: '600' },
  small: { fontSize: 13, lineHeight: 19 },
  body: { fontSize: 16, lineHeight: 24 },
  headline: { fontFamily: serif, fontSize: 27, fontWeight: '600' },
  screenTitle: { fontFamily: serif, fontSize: 42, fontWeight: '600' },
});

export function Screen({ children, tab = false }: PropsWithChildren<{ tab?: boolean }>) {
  const { colors } = usePalette();
  return <SafeAreaView style={[layout.flex, { backgroundColor: colors.background }]}
    edges={tab ? ['top', 'left', 'right'] : ['left', 'right', 'bottom']}>
    {children}
  </SafeAreaView>;
}

export function Page({ children }: PropsWithChildren) {
  return <ScrollView keyboardShouldPersistTaps="handled" contentContainerStyle={[layout.page, layout.stack, { paddingTop: 20 }]}>
    {children}
  </ScrollView>;
}

export function ScreenHeading({ title, note, actions }: { title: string; note?: string; actions?: ReactNode }) {
  const { colors } = usePalette();
  return <View style={[styles.heading, { borderBottomColor: colors.line }]}>
    <View style={layout.row}>
      <Image source={require('../assets/icon.png')} accessible={false} style={styles.logo} />
      <Text style={[layout.label, { color: colors.text }]}>{copy.appName}</Text>
    </View>
    <Text accessibilityRole="header" style={[layout.screenTitle, { color: colors.text }]}>{title}</Text>
    {note && <Text style={[layout.small, { color: colors.muted }]}>{note}</Text>}
    {actions && <View style={[layout.wrap, styles.headingActions]}>{actions}</View>}
  </View>;
}

type ButtonProps = {
  label: string;
  onPress: () => void;
  icon?: LucideIcon;
  tone?: 'primary' | 'quiet' | 'danger';
  busy?: boolean;
  disabled?: boolean;
};

export function Button({ label, onPress, icon: Icon, tone = 'primary', busy, disabled }: ButtonProps) {
  const { colors } = usePalette();
  const primary = tone === 'primary';
  const color = primary ? colors.onAccent : tone === 'danger' ? colors.danger : colors.text;
  return <Pressable accessibilityRole="button" accessibilityState={{ disabled: !!(disabled || busy), busy }}
    onPress={onPress} disabled={disabled || busy}
    style={({ pressed }) => [
      styles.button, { backgroundColor: primary ? colors.accent : colors.paper, borderColor: primary ? colors.accent : colors.line },
      (pressed || disabled) && styles.dim,
    ]}>
    {busy ? <ActivityIndicator color={color} /> : Icon && <Icon size={18} color={color} accessible={false} />}
    <Text style={[layout.label, { color, flexShrink: 1 }]}>{label}</Text>
  </Pressable>;
}

export function IconButton({ label, icon: Icon, onPress, active, disabled }: {
  label: string; icon: LucideIcon; onPress: () => void; active?: boolean; disabled?: boolean;
}) {
  const { colors } = usePalette();
  return <Pressable accessibilityRole="button" accessibilityLabel={label}
    accessibilityState={{ selected: active, disabled }} onPress={onPress} disabled={disabled}
    style={({ pressed }) => [styles.iconButton, { backgroundColor: active || pressed ? colors.soft : 'transparent' }, disabled && styles.dim]}>
    <Icon size={22} color={active ? colors.accent : colors.text} accessible={false} />
  </Pressable>;
}

export function Segments<T extends string>({ tabs, value, onChange }: {
  tabs: { id: T; label: string }[]; value: T; onChange: (id: T) => void;
}) {
  const { colors } = usePalette();
  return <View accessibilityRole="tablist" style={[styles.segments, { borderColor: colors.line }]}>
    {tabs.map(({ id, label }) => <Pressable key={id} accessibilityRole="tab"
      accessibilityState={{ selected: value === id }} onPress={() => onChange(id)}
      style={({ pressed }) => [styles.segment, { backgroundColor: value === id || pressed ? colors.soft : 'transparent' }]}>
      <Text style={[layout.label, { color: value === id ? colors.text : colors.muted, textAlign: 'center' }]}>{label}</Text>
    </Pressable>)}
  </View>;
}

export function EmptyState({ icon: Icon, title, body }: { icon: LucideIcon; title: string; body: string }) {
  const { colors } = usePalette();
  return <View style={[styles.empty, { backgroundColor: colors.paper, borderColor: colors.line }]}>
    <View style={[styles.emptyIcon, { backgroundColor: colors.soft }]}><Icon size={26} color={colors.accent} accessible={false} /></View>
    <Text accessibilityRole="header" style={[layout.headline, { color: colors.text }]}>{title}</Text>
    <Text style={[layout.body, { color: colors.muted }]}>{body}</Text>
  </View>;
}

export function Notice({ text, error = false }: { text: string; error?: boolean }) {
  const { colors } = usePalette();
  return <Text accessibilityRole={error ? 'alert' : undefined} accessibilityLiveRegion="polite"
    style={[layout.small, styles.notice, { color: error ? colors.danger : colors.text, backgroundColor: error ? colors.dangerSurface : colors.soft }]}>
    {text}
  </Text>;
}

export function Field({ label, ...props }: { label: string } & TextInputProps) {
  const { colors } = usePalette();
  return <View style={{ gap: 7 }}>
    <Text style={[layout.label, { color: colors.text }]}>{label}</Text>
    <TextInput {...props} accessibilityLabel={label} placeholderTextColor={colors.muted}
      style={[styles.input, { color: colors.text, backgroundColor: colors.paper, borderColor: colors.line }, props.style]} />
  </View>;
}

export function Loading({ message = copy.busy }: { message?: string }) {
  const { colors } = usePalette();
  return <View style={[layout.row, { paddingVertical: 14 }]} accessibilityLiveRegion="polite">
    <ActivityIndicator color={colors.accent} />
    <Text style={[layout.body, { color: colors.muted }]}>{message}</Text>
  </View>;
}

const styles = StyleSheet.create({
  heading: { paddingTop: 15, paddingBottom: 20, gap: 6, borderBottomWidth: StyleSheet.hairlineWidth, marginBottom: 18 },
  headingActions: { marginTop: 10 },
  logo: { width: 28, height: 28, borderRadius: 7 },
  button: {
    flexDirection: 'row', alignItems: 'center', justifyContent: 'center',
    minHeight: 48, paddingHorizontal: 14, paddingVertical: 8, gap: 8, borderRadius: 9, borderWidth: 1,
  },
  iconButton: { width: 46, height: 46, alignItems: 'center', justifyContent: 'center', borderRadius: 9 },
  segments: { flexDirection: 'row', borderWidth: 1, borderRadius: 10, padding: 3, gap: 2 },
  segment: { flex: 1, minHeight: 44, padding: 6, alignItems: 'center', justifyContent: 'center', borderRadius: 7 },
  empty: { borderWidth: 1, borderRadius: 14, padding: 22, gap: 12, marginTop: 14 },
  emptyIcon: { width: 49, height: 49, borderRadius: 11, alignItems: 'center', justifyContent: 'center' },
  input: { fontSize: 16, minHeight: 50, paddingHorizontal: 13, paddingVertical: 10, borderWidth: 1, borderRadius: 9 },
  notice: { paddingVertical: 11, paddingHorizontal: 13, borderRadius: 8 },
  dim: { opacity: 0.65 },
});
