// tmux-style prefix mode, migrated from herdr's `prefix = ["ctrl+b", "ctrl+ㅠ"]`.
// keys.json binds ctrl-b to "mod.open:prefix"; this popup takes the next key,
// runs its command and closes. Esc or an unknown key cancels.
// ctrl-b twice sends a literal ctrl+b to the focused terminal.
// Korean 2-set jamo are mapped to their QWERTY keys, so prefix keys work with the IME on.

import { View, div } from "gpui-kit";
import { h_flex } from "gpui-base";
import { run, hc, focused, props, closeOverlay, theme } from "herdr";

const ACTIONS = {
  "?": ["app.help"],
  "shift-/": ["app.help"],
  s: ["app.settings"],
  w: ["mod.open", { name: "picker" }],
  g: ["mod.open", { name: "picker" }],
  "shift-n": ["space.new"],
  "shift-w": ["space.rename"],
  "shift-d": ["space.close"],
  c: ["tab.new"],
  "shift-t": ["tab.rename"],
  p: ["tab.prev"],
  n: ["tab.next"],
  "shift-x": ["tab.close"],
  h: ["pane.go", { direction: "left" }],
  j: ["pane.go", { direction: "down" }],
  k: ["pane.go", { direction: "up" }],
  l: ["pane.go", { direction: "right" }],
  left: ["pane.go", { direction: "left" }],
  down: ["pane.go", { direction: "down" }],
  up: ["pane.go", { direction: "up" }],
  right: ["pane.go", { direction: "right" }],
  v: ["pane.split", { direction: "right" }],
  "-": ["pane.split", { direction: "down" }],
  minus: ["pane.split", { direction: "down" }],
  x: ["pane.close"],
  z: ["pane.zoom"],
  b: ["layout.toggle", { id: "sidebar" }],
  t: ["terminal.open", { argv: ["zsh"], place: "down" }],
  space: ["app.palette"],
  " ": ["app.palette"],
};
for (let i = 1; i <= 9; i++) ACTIONS[String(i)] = ["tab.select", { index: i }];

// Korean 2-set layout → QWERTY key.
const HANGUL = {
  ㅂ: "q", ㅈ: "w", ㄷ: "e", ㄱ: "r", ㅅ: "t", ㅛ: "y", ㅕ: "u", ㅑ: "i", ㅐ: "o", ㅔ: "p",
  ㅁ: "a", ㄴ: "s", ㅇ: "d", ㄹ: "f", ㅎ: "g", ㅗ: "h", ㅓ: "j", ㅏ: "k", ㅣ: "l",
  ㅋ: "z", ㅌ: "x", ㅊ: "c", ㅍ: "v", ㅠ: "b", ㅜ: "n", ㅡ: "m",
  ㅃ: "q", ㅉ: "w", ㄸ: "e", ㄲ: "r", ㅆ: "t", ㅒ: "o", ㅖ: "p",
};
const SHIFTED_HANGUL = new Set(["ㅃ", "ㅉ", "ㄸ", "ㄲ", "ㅆ", "ㅒ", "ㅖ"]);

function spec(e) {
  let key = (e.key || "").toLowerCase();
  let shift = !!e.modifiers?.shift;
  if (HANGUL[e.key]) {
    shift = shift || SHIFTED_HANGUL.has(e.key);
    key = HANGUL[e.key];
  }
  if (e.modifiers?.control && key === "b") return "ctrl-b";
  if (e.modifiers?.control || e.modifiers?.platform || e.modifiers?.alt) return null;
  return shift ? `shift-${key}` : key;
}

export default class Prefix extends View {
  init(_props, cx) {
    this.handle = cx.focus_handle();
    this.handle.focus();
  }

  close() {
    const o = props()?.overlay;
    if (o) closeOverlay(o);
  }

  press(e) {
    if (["shift", "control", "alt", "platform", "function"].includes(e.key)) return false;
    const s = spec(e);
    this.close();
    if (s === "ctrl-b") {
      const t = focused()?.terminal;
      if (t) hc(["keys", t, "ctrl+b"]);
      return true;
    }
    const a = s && ACTIONS[s];
    if (a) run(a[0], a[1] ?? {});
    return true;
  }

  render(_cx) {
    const t = theme();
    return h_flex()
      .id("prefix")
      .track_focus(this.handle)
      .on_key_down((e, cx) => {
        if (this.press(e)) cx.stop_propagation();
      })
      .size_full()
      .items_center()
      .justify_center()
      .gap(6)
      .rounded(8)
      .border_1()
      .border_color(t.accent)
      .bg(t.popover)
      .text_size(13)
      .child(div().text_color(t.accent).child("ctrl+b"))
      .child(div().text_color(t.muted).child("…"));
  }
}
