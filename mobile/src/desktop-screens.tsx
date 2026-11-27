import { useFocusEffect, useNavigation, usePreventRemove } from '@react-navigation/native';
import type { NativeStackNavigationProp, NativeStackScreenProps } from '@react-navigation/native-stack';
import { ArrowUpRight, BookOpen, HardDrive, Link2, Monitor, Trash2 } from 'lucide-react-native';
import { useCallback, useRef, useState } from 'react';
import { Alert, Keyboard, Text, View } from 'react-native';
import appConfig from '../app.json';
import { forgetConnection, getConnection, storeConnection } from './connection-storage';
import { copy, errorMessage } from './copy';
import { type Connection, listDesktopSites, normalizeConnection } from './desktop';
import type { Routes } from './routes';
import { usePalette } from './theme';
import { Button, Field, layout, Loading, Notice, Page, Screen, ScreenHeading } from './ui';

export function SettingsScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<Routes>>();
  const [connection, setConnection] = useState<Connection | null>(null);
  const [error, setError] = useState('');
  const { colors } = usePalette();
  useFocusEffect(useCallback(() => {
    let visible = true;
    void getConnection().then((value) => {
      if (visible) {
        setConnection(value);
        setError('');
      }
    }).catch((cause) => {
      if (visible) setError(errorMessage(cause));
    });
    return () => { visible = false; };
  }, []));

  return <Screen tab><Page>
    <ScreenHeading title={copy.settings} />
    {error && <Notice text={error} error />}
    <Button label={copy.desktop} icon={Monitor} tone="quiet" onPress={() => navigation.navigate('Desktop')} />
    <Text style={[layout.small, { color: colors.muted }]}>
      {connection ? connection.origin : copy.disconnected}
    </Text>
    <Button label={copy.manageFeeds} icon={BookOpen} tone="quiet" onPress={() => navigation.navigate('Subscriptions')} />
    <View style={{ gap: 8, marginTop: 10 }}>
      <View style={layout.row}>
        <HardDrive size={18} color={colors.accent} accessible={false} />
        <Text accessibilityRole="header" style={[layout.label, { color: colors.text }]}>{copy.offlineStorage}</Text>
      </View>
      <Text style={[layout.body, { color: colors.muted }]}>{copy.storageHint}</Text>
    </View>
    <Text style={[layout.small, { color: colors.muted, marginTop: 18 }]}>{copy.appName} {appConfig.expo.version}</Text>
  </Page></Screen>;
}

export function DesktopScreen({ navigation }: NativeStackScreenProps<Routes, 'Desktop'>) {
  const { colors } = usePalette();
  const [connection, setConnection] = useState<Connection | null>(null);
  const [origin, setOrigin] = useState('');
  const [username, setUsername] = useState('Planet');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(true);
  const [working, setWorking] = useState(false);
  const [error, setError] = useState('');
  const active = useRef(false);
  usePreventRemove(working, () => {});
  useFocusEffect(useCallback(() => {
    let visible = true;
    void getConnection().then((value) => {
      if (!visible) return;
      setConnection(value);
      if (value) {
        setOrigin(value.origin);
        setUsername(value.username);
      }
    }).catch((cause) => {
      if (visible) setError(errorMessage(cause));
    }).finally(() => {
      if (visible) setLoading(false);
    });
    return () => { visible = false; };
  }, []));

  async function connect() {
    if (active.current) return;
    active.current = true;
    setWorking(true);
    setError('');
    Keyboard.dismiss();
    try {
      const normalized = normalizeConnection(origin, username, password);
      await listDesktopSites(normalized);
      await storeConnection(normalized);
      setOrigin(normalized.origin);
      setConnection(normalized);
      setPassword('');
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      active.current = false;
      setWorking(false);
    }
  }

  async function forget() {
    try {
      await forgetConnection();
      setConnection(null);
      setOrigin('');
      setUsername('Planet');
      setPassword('');
      setError('');
    } catch (cause) {
      setError(errorMessage(cause));
    }
  }

  return <Screen><Page>
    {loading ? <Loading /> : <>
      {connection && <Notice text={copy.connected + ' · ' + connection.origin} />}
      <Text style={[layout.body, { color: colors.muted }]}>{copy.desktopHint}</Text>
      <Field label={copy.endpoint} placeholder="https://kinpage.example.org" keyboardType="url"
        autoCapitalize="none" autoCorrect={false} maxLength={256} editable={!working} value={origin} onChangeText={setOrigin} />
      <Field label={copy.username} autoCapitalize="none" autoCorrect={false} maxLength={80}
        editable={!working} value={username} onChangeText={setUsername} />
      <Field label={copy.password} secureTextEntry autoCapitalize="none" autoCorrect={false} maxLength={256}
        editable={!working} value={password} onChangeText={setPassword} />
      {error && <Notice text={error} error />}
      <Button label={working ? copy.checkingConnection : copy.saveConnection} icon={Link2} onPress={connect}
        disabled={!origin.trim() || !username.trim() || !password} busy={working} />
      {connection && <>
        <Button label={copy.back} icon={ArrowUpRight} tone="quiet" disabled={working} onPress={() => navigation.goBack()} />
        <Button label={copy.forgetConnection} icon={Trash2} tone="danger" disabled={working} onPress={() => {
          Alert.alert(copy.forgetConnection, copy.forgetBody, [
            { text: copy.cancel, style: 'cancel' },
            { text: copy.remove, style: 'destructive', onPress: forget },
          ]);
        }} />
      </>}
    </>}
  </Page></Screen>;
}
