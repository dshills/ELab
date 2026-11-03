const paths = {
 search:'M20 20l-5-5M17 10a7 7 0 1 1-14 0 7 7 0 0 1 14 0',
 'panel-left':'M3 4h18v16H3zM9 4v16', 'panel-right':'M3 4h18v16H3zM15 4v16',
 moon:'M20 14A9 9 0 0 1 10 3a9 9 0 1 0 10 11',sun:'M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8M12 2v2m0 16v2M2 12h2m16 0h2M5 5l1 1m12 12 1 1M5 19l1-1M18 6l1-1',
 folder:'M3 7V5h6l2 3h10v12H3z', download:'M12 3v12m-5-5 5 5 5-5M4 17v4h16v-4',
 cursor:'m5 3 14 10-7 1-3 7z',wire:'M4 5h6v14h10M2 5a2 2 0 1 0 4 0 2 2 0 0 0-4 0M18 19a2 2 0 1 0 4 0 2 2 0 0 0-4 0',
 hand:'M8 13V5a2 2 0 0 1 4 0v6-8a2 2 0 0 1 4 0v8-6a2 2 0 0 1 4 0v10c0 5-4 7-7 7-5 0-6-5-9-8a2 2 0 0 1 3-2l2 2',
 undo:'M8 4 3 9l5 5M3 9h11a6 6 0 0 1 0 12',redo:'m16 4 5 5-5 5m5-5H10a6 6 0 0 0 0 12',
 fit:'M8 3H3v5m13-5h5v5M3 16v5h5m8 0h5v-5M8 8h8v8H8z',restart:'M4 10a8 8 0 1 1 2 8M4 3v7h7',
 step:'m5 5 11 7-11 7zM19 5v14',pause:'M8 5v14M16 5v14',play:'m7 4 13 8-13 8z',
 minus:'M5 12h14',plus:'M12 5v14M5 12h14','chevron-down':'m6 9 6 6 6-6','arrow-right':'M4 12h16m-6-6 6 6-6 6',
 close:'m6 6 12 12M6 18 18 6',circuits:'M4 12h4m8 0h4M8 8h8v8H8zM12 4v4m0 8v4',
 help:'M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20M9 8a3 3 0 1 1 5 2c-2 1-2 2-2 4m0 3v.1',
 rotate:'M4 10a8 8 0 1 1 2 8M4 4v6h6',trash:'M4 6h16M9 6V3h6v3M6 6l1 15h10l1-15M10 10v7m4-7v7',
 waveform:'M2 12h4l3-8 6 16 3-8h4',check:'m5 12 4 4L19 6',file:'M5 3h9l5 5v13H5zM14 3v6h5',
 spark:'m12 2-3 7H2l6 5-2 8 6-5 6 5-2-8 6-5h-7z',settings:'M4 7h16M7 4v6m-3 7h16m-5-3v6',
 eye:'M2 12s4-7 10-7 10 7 10 7-4 7-10 7S2 12 2 12m7 0a3 3 0 1 0 6 0 3 3 0 0 0-6 0',
 copy:'M8 8h13v13H8zM16 8V3H3v13h5',pin:'m8 3 8 0-1 7 4 4H5l4-4zM12 14v8',
};
export function icon(name, cls=''){return `<svg class="icon ${cls}" viewBox="0 0 24 24" aria-hidden="true"><path d="${paths[name]||paths.circuits}"/></svg>`;}
export function hydrateIcons(root=document){root.querySelectorAll('[data-icon]').forEach(el=>el.replaceWith(Object.assign(document.createElement('span'),{className:el.className,innerHTML:icon(el.dataset.icon)})));}
