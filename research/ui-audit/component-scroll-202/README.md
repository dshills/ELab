# Mouse scrolling and scrollbar evidence

All images were returned by CUA, then saved unchanged as JPEG files. Before
captures identify /201; preview captures identify the first event-location fix,
which precedes the separate scrollbar wheel-direction fix.

- 01 /201: wheel down over rows left the list at its first components.
- 02 /201: dragging the thumb down left it at the top.
- 03 preview: wheel down exposes the final SPST switch and bottom thumb.
- 04 preview: wheel up restores DC source and top thumb.
- 05 preview: thumb drag down exposes SPST switch at the bottom.
- 06 preview: intermediate reverse-drag capture had a different image size and
  still showed the bottom. It is not qualifying evidence.
- 07 preview: refreshed view confirms the reverse drag returned to the top.
- 08 preview: mouse-focused Search and typed mosfet yield two matching parts.

Final /202 packaged-app captures are recorded separately with explicit captions.
All browser actions used empty documents; no owner circuit was edited.

Final packaged ELab 0.181.1 /202 (native SHA-256
cb1b0ddebb6c86d1b5e88a5749bd0aedb675bd7c84f5162a9a814a8414b86c88):

- 09: All components at the top after a normal product launch.
- 10: wheel down directly over the scrollbar exposes SPST switch at the bottom.
- 11: wheel up over the scrollbar returns to DC source at the top.
- 12: wheel down over the component rows reaches the same bottom position.
- 13: dragging the bottom thumb halfway upward exposes Sine source, finite pulse,
  square, MOSFETs and NPN in the middle; the thumb moves proportionally.
- 14: continuing the upward thumb drag restores the first components.
- 15: a downward thumb drag reaches the last component again.
- 16: wheel up over the rows restores the first component again.
- 17: mouse selection chose Resistor, a coordinate click focused Search, and
  Select then canceled placement. The final normal app remains open in All,
  at the top, with zero components/wires and no dirty marker. The AX capture
  accompanies this image.

Search and category stay above the list throughout. Product launches retain
normal recovery and appearance behavior; the preliminary isolated preview used
session-only flags and was closed. No pointer event replay outside CUA was used.
