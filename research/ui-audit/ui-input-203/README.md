# UI mouse and scrolling evidence

CUA returned these unmodified window captures from ELab 0.181.1 /202. They are
baseline/first-pass evidence, before the fixes in /203. Actions and outcomes are
listed in [the maintenance notes](../../78-ui-input-checks.md).

1. Starting clean empty circuit, 2214×1264.
2. Compact Inspect rail after wheel-down and thumb back to top.
3. Simulate rail wheel reached statistics in the flasher fixture.
4. Wheel over scope plot exposed the cursor controls.
5. Scope A stayed at 3.944961832061069 s after a short drag to the right.
6. A secondary click incorrectly changed scope A to 4.502213740458015 s.
7. Short divider drag left its accepted fraction and layout unchanged.
8. Document wheel reached recovery; existing recovery copies were untouched.
9. Preview margin +/− and settings thumb worked; no file or print submitted.
10. Potentiometer drag accepted about 75.1%; both synthetic edits were undone.
11. Recorded sample navigation and plot scrolling worked.
12. Sweep click selected point 3; short drag left it at point 3.
13–14. Test cleanup/resize attempts. These are clean empty documents, but the
captured size is 1221×969 rather than the original size; neither capture proves
exact native-window movement or original-size restoration.

Final /203 packaged-app captures and captions are appended separately after
verification; baseline captures do not qualify repaired behavior.
