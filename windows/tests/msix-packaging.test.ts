import { expect, test } from "bun:test";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { join, resolve } from "node:path";

const windowsDir = resolve(import.meta.dir, "..");
const testScratchRoot = resolve(windowsDir, ".mygo/msix-tests");
const msixScript = resolve(windowsDir, "scripts/build-msix.ps1");
const msixConfigPath = resolve(windowsDir, "msix/msix.config.json");
const msixTemplatePath = resolve(windowsDir, "msix/AppxManifest.template.xml");
const releaseJsonPath = resolve(windowsDir, "release.json");

const requiredPayloadFiles = [
  "Planet Windows.exe",
  "ipfs.exe",
  "icon.png",
  "kubo-LICENSE.txt",
  "kubo-LICENSE-MIT.txt",
  "kubo-LICENSE-APACHE.txt",
  "LICENSE-PLANET.txt",
  "THIRD-PARTY-NOTICES.txt",
  "KUBO-BUILDINFO.txt",
  "KUBO-DEPENDENCY-LICENSES.txt",
];

const expectedAssets = [
  "StoreLogo.png",
  "Square44x44Logo.png",
  "Square44x44Logo.targetsize-48.png",
  "Square44x44Logo.targetsize-48_altform-unplated.png",
  "SmallTile.png",
  "Square150x150Logo.png",
  "Wide310x150Logo.png",
  "LargeTile.png",
  "SplashScreen.png",
];

function makeTempDir(prefix: string): string {
  mkdirSync(testScratchRoot, { recursive: true });
  return mkdtempSync(join(testScratchRoot, prefix));
}

function createSyntheticPayload(): string {
  const dir = makeTempDir("planet-msix-payload-");
  for (const name of requiredPayloadFiles) {
    writeFileSync(join(dir, name), `synthetic:${name}\n`, "utf8");
  }
  return dir;
}

test("MSIX config and manifest template declare required desktop capabilities", () => {
  const config = JSON.parse(readFileSync(msixConfigPath, "utf8"));
  expect(config.packageName).toMatch(/^[A-Za-z0-9.-]{3,50}$/);
  expect(config.publisher).toMatch(/^CN=.+/);
  expect(config.languages).toContain("zh-CN");
  expect(config.languages).toContain("en-US");

  const template = readFileSync(msixTemplatePath, "utf8");
  expect(template).toContain('Name="internetClient"');
  expect(template).toContain('Name="privateNetworkClientServer"');
  expect(template).toContain('Name="runFullTrust"');
  expect(template).toContain('Executable="Planet Windows.exe"');
});

test("build-msix.ps1 stages manifest, payload files, and generated PNG assets", () => {
  const payloadDir = createSyntheticPayload();
  const stageDir = makeTempDir("planet-msix-stage-");
  try {
    const result = Bun.spawnSync([
      "powershell",
      "-NoProfile",
      "-ExecutionPolicy",
      "Bypass",
      "-File",
      msixScript,
      "-PayloadDir",
      payloadDir,
      "-StageDir",
      stageDir,
      "-StageOnly",
      "-SkipVersionProbe",
    ]);
    if (result.exitCode !== 0) {
      throw new Error(result.stderr.toString() || result.stdout.toString());
    }

    const release = JSON.parse(readFileSync(releaseJsonPath, "utf8"));
    const expectedMsixVersion = `${release.version.split("-")[0]}.0`;
    const manifest = readFileSync(join(stageDir, "AppxManifest.xml"), "utf8");

    expect(manifest).not.toContain("{{");
    expect(manifest).toContain(`Version="${expectedMsixVersion}"`);
    expect(manifest).toContain('ProcessorArchitecture="x64"');
    expect(manifest).toContain('<Resource Language="zh-CN" />');
    expect(manifest).toContain('<Resource Language="zh-TW" />');
    expect(manifest).toContain('<Resource Language="en-US" />');

    for (const fileName of requiredPayloadFiles) {
      expect(existsSync(join(stageDir, fileName))).toBe(true);
    }
    for (const assetName of expectedAssets) {
      const assetPath = join(stageDir, "Assets", assetName);
      expect(existsSync(assetPath)).toBe(true);
      const bytes = readFileSync(assetPath);
      // Verify PNG signature (89 50 4E 47)
      expect(bytes[0]).toBe(0x89);
      expect(bytes[1]).toBe(0x50);
      expect(bytes[2]).toBe(0x4e);
      expect(bytes[3]).toBe(0x47);
    }
  } finally {
    rmSync(payloadDir, { recursive: true, force: true });
    rmSync(stageDir, { recursive: true, force: true });
  }
});

test("build-msix.ps1 -Store rejects placeholder publisher and non-zero revision", () => {
  const payloadDir = createSyntheticPayload();
  const stageDir = makeTempDir("planet-msix-store-");
  try {
    const placeholderResult = Bun.spawnSync([
      "powershell",
      "-NoProfile",
      "-ExecutionPolicy",
      "Bypass",
      "-File",
      msixScript,
      "-PayloadDir",
      payloadDir,
      "-StageDir",
      stageDir,
      "-Store",
      "-AllowPrerelease",
      "-StageOnly",
      "-SkipVersionProbe",
    ]);
    expect(placeholderResult.exitCode).not.toBe(0);
    expect(placeholderResult.stderr.toString()).toContain("Partner Center Publisher ID");

    const badRevisionResult = Bun.spawnSync([
      "powershell",
      "-NoProfile",
      "-ExecutionPolicy",
      "Bypass",
      "-File",
      msixScript,
      "-PayloadDir",
      payloadDir,
      "-StageDir",
      stageDir,
      "-Publisher",
      "CN=12345678-1234-1234-1234-1234567890AB",
      "-MsixVersion",
      "0.0.1.1",
      "-Store",
      "-AllowPrerelease",
      "-StageOnly",
      "-SkipVersionProbe",
    ]);
    expect(badRevisionResult.exitCode).not.toBe(0);
    expect(badRevisionResult.stderr.toString()).toContain("requires it to be 0");
  } finally {
    rmSync(payloadDir, { recursive: true, force: true });
    rmSync(stageDir, { recursive: true, force: true });
  }
});
