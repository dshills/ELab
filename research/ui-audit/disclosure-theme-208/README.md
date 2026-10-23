# Technical-details session colors

October 23, 2026. [Repair and qualification scope](../../85-disclosure-theme.md).
Actual Vellum manual-window renders of a 320×360 inspector, using managed Noto
Sans test fonts. These are offscreen renders, not macOS desktop captures.

1. `01-before-dark.png`: retained /207 render; light surface and blue heading.
2. `02-surface-only-dark.png`: the app's surface fix with the old elab6 driver;
   the blue heading remains and the consumer glyph-paint regression fails.
3. `03-final-dark.png`: elab7 heading inherits light ink on the session surface.
4. `04-final-light.png`: the same disclosure uses dark ink on the light surface.

The consumer regression also physically opens the disclosure, then collapses
through the shared public agent command without changing accepted circuit,
document, time or history. The driver regression covers simultaneous independent
palettes and post-mount updates. Neither proves complete native, VoiceOver,
contrast/focus-state or IME acceptance. No image was edited or recolored.
