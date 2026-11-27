import { Platform, useColorScheme } from 'react-native';

const light = {
  background: '#FBFAF7', paper: '#FFFFFF', text: '#333432', muted: '#626760',
  line: '#DDDDD4', accent: '#4F6B5B', soft: '#E9EFE8',
  onAccent: '#FFFFFF', danger: '#943C35', dangerSurface: '#F8EAE5',
};
const dark: typeof light = {
  background: '#1C211E', paper: '#252C27', text: '#F0EEE5', muted: '#B8BFB4',
  line: '#434C43', accent: '#AFC9B2', soft: '#334338',
  onAccent: '#1D2B20', danger: '#F5AAA0', dangerSurface: '#462D29',
};

export const serif = Platform.select({ ios: 'Georgia', android: 'serif', default: 'serif' });

export function usePalette() {
  const isDark = useColorScheme() === 'dark';
  return { colors: isDark ? dark : light, isDark };
}
