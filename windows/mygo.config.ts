import { defineConfig } from "mygo-cli";
import release from "./release.json";

export default defineConfig({
  name: "Kinpage",
  identifier: "dev.puremixlab.kinpage",
  version: release.version,
  // `mygo build` runs buildCommand before compiling the Native UI app.
  buildCommand: "powershell -NoProfile -File scripts/prepare-notices.ps1",
  out: "build",
  windows: { installerInclude: "scripts/installer-safe-uninstall.nsh" },
});
