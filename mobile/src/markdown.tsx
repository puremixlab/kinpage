import type { ReactNode } from 'react';
import { useMemo } from 'react';
import { Alert, Linking, type TextStyle, View } from 'react-native';
import { Renderer, useMarkdown } from 'react-native-marked';
import { copy } from './copy';
import { httpsURL } from './network.ts';
import { serif, usePalette } from './theme';

export function openLink(value: string) {
  const url = httpsURL(value);
  if (url) void Linking.openURL(url.href).catch(() => Alert.alert(copy.linkFailed));
}

class ReadingRenderer extends Renderer {
  override link(children: string | ReactNode[], href: string, style?: TextStyle) {
    const url = httpsURL(href);
    return url ? super.link(children, url.href, style) : this.text(children, style);
  }

  override image(_uri: string, alt?: string) {
    return this.text('[' + (alt || copy.markdownImage) + ']');
  }

  override linkImage(href: string, _imageUrl: string, alt?: string) {
    return this.link('[' + (alt || copy.markdownImage) + ']', href);
  }
}

export function MarkdownBody({ value, baseUrl }: { value: string; baseUrl?: string }) {
  const { colors, isDark } = usePalette();
  const renderer = useMemo(() => new ReadingRenderer(), []);
  const styles = useMemo(() => ({
    text: { fontSize: 17, lineHeight: 27, color: colors.text },
    h1: { fontFamily: serif, color: colors.text },
    h2: { fontFamily: serif, color: colors.text },
    link: { color: colors.accent },
    code: { backgroundColor: colors.soft },
    codeText: { color: colors.text },
    hr: { borderColor: colors.line, backgroundColor: colors.line },
  }), [colors]);
  const body = useMarkdown(value, { renderer, baseUrl, styles, colorScheme: isDark ? 'dark' : 'light' });
  return <View>{body}</View>;
}
