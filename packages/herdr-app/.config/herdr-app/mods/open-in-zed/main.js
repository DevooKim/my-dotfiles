// Opens the focused terminal's folder in Zed, then closes itself.
// Bound to alt-i in keys.json as "mod.open:open-in-zed"
// (migrated from herdr's `alt+i` shell command `zed .`).

import { View, div } from "gpui-kit";
import { exec, focused, notify, props, closeOverlay } from "herdr";

export default class OpenInZed extends View {
  init(_props, cx) {
    const cwd = focused()?.cwd;
    const overlay = props()?.overlay;
    cx.spawn(async (_cx) => {
      if (overlay) closeOverlay(overlay);
      if (!cwd) {
        notify({ title: "Open in Zed", body: "The focused terminal has no folder." });
        return;
      }
      try {
        const r = await exec("zed", [cwd], { timeout: 10000 });
        if (r.code !== 0 && r.code != null) {
          notify({ title: "Open in Zed failed", body: (r.stderr || r.stdout || `exit ${r.code}`).trim() });
        }
      } catch (e) {
        notify({ title: "Open in Zed failed", body: String(e) });
      }
    });
  }

  render(_cx) {
    return div();
  }
}
